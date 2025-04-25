package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"runbin/internal/model"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func monitorMemory(ctx context.Context, cli *client.Client, containerID string, maxMem *uint64) {
	// 创建独立上下文用于内存监控
	memCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	ticker := time.NewTicker(64 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-memCtx.Done():
			return
		case <-ticker.C:
			// 检查容器是否仍在运行
			_, err := cli.ContainerInspect(memCtx, containerID)
			if err != nil {
				return
			}

			statsResp, err := cli.ContainerStats(memCtx, containerID, false)
			if err != nil {
				continue // 忽略临时错误
			}

			var statsJSON container.StatsResponse
			if err := json.NewDecoder(statsResp.Body).Decode(&statsJSON); err != nil {
				statsResp.Body.Close()
				continue
			}
			statsResp.Body.Close()

			if statsJSON.MemoryStats.Stats != nil {
				if currentMem, ok := statsJSON.MemoryStats.Stats["anon"]; ok {
					if currentMem > *maxMem {
						*maxMem = currentMem
					}
				}
			}
		}
	}
}

func (w *Worker) RunCppTask(ctx context.Context, task *model.Paste, cli *client.Client) error {
	// Create temporary workspace
	tmpDir, err := os.MkdirTemp("", "cpp_compile_")
	if err != nil {
		return fmt.Errorf("create temp dir error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write user code to main.cpp
	codePath := filepath.Join(tmpDir, "main.cpp")
	if err := os.WriteFile(codePath, []byte(task.Code), 0644); err != nil {
		return fmt.Errorf("write code file error: %v", err)
	}

	// Get execution limits from config
	timeout := time.Duration(w.cfg.Limit.Time) * time.Second
	compliteCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	hostConfig := &container.HostConfig{
		Binds: []string{tmpDir + ":/app"},
		Resources: container.Resources{
			Memory:   int64(w.cfg.Limit.Memory * 1024 * 1024),
			CPUQuota: int64(w.cfg.Limit.Cpu * 100000),
		},
		NetworkMode: "none",
	}

	// Create compile container configuration
	resp, err := cli.ContainerCreate(compliteCtx, &container.Config{
		Image: "gcc:14",
		Cmd:   []string{"sh", "-c", "g++ -std=c++20 /app/main.cpp -o /app/output > /app/compile.txt 2>&1"},
	}, hostConfig, nil, nil, filepath.Base(tmpDir)+"_builder")
	if err != nil {
		return fmt.Errorf("create compile container error: %v", err)
	}
	defer cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{
		Force: true,
	})

	// Start compile container execution
	if err := cli.ContainerStart(compliteCtx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start compile container: %v", err)
	}

	// Wait for compile container completion
	statusCh, errCh := cli.ContainerWait(compliteCtx, resp.ID, container.WaitConditionNotRunning)
	// Handle compile container execution results
	select {
	case <-statusCh:
		// Get compile container detailed state
		containerState, err := cli.ContainerInspect(compliteCtx, resp.ID)
		if err != nil {
			return fmt.Errorf("container state inspection error: %v", err)
		}

		// Read compilation log
		if logData, err := os.ReadFile(filepath.Join(tmpDir, "compile.txt")); err == nil {
			task.CompileLog = string(logData)
		}

		// Non-zero exit code indicates compilation failure
		if containerState.State.ExitCode != 0 {
			task.Status = model.StatusCompileError
			return nil
		}
	case err := <-errCh:
		return err
	case <-compliteCtx.Done():
		task.Status = model.StatusCompileError
		task.CompileLog = "Compile process exceeded time limit"
		return nil
	}

	// Write input to input.txt
	inputPath := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(inputPath, []byte(task.Stdin), 0644); err != nil {
		return fmt.Errorf("write input file error: %v", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create runner container configuration
	resp, err = cli.ContainerCreate(runCtx, &container.Config{
		Image: "gcc:14",
		Cmd:   []string{"sh", "-c", "sh -c \"/app/output < /app/input.txt > /app/stdout.txt\" > /app/stderr.txt 2>&1"},
	}, hostConfig, nil, nil, filepath.Base(tmpDir)+"_runner")
	if err != nil {
		return fmt.Errorf("create runner container error: %v", err)
	}
	defer cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{
		Force: true,
	})

	// Start runner container execution
	if err := cli.ContainerStart(runCtx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start runner container: %v", err)
	}

	// 启动内存监控协程（使用独立上下文）
	var maxMem uint64
	go monitorMemory(context.Background(), cli, resp.ID, &maxMem)

	// 等待容器完成
	statusCh, errCh = cli.ContainerWait(runCtx, resp.ID, container.WaitConditionNotRunning)

	// 处理执行结果
	select {
	case <-statusCh:
		// Get container detailed state
		containerState, err := cli.ContainerInspect(runCtx, resp.ID)
		if err != nil {
			return fmt.Errorf("container state inspection error: %v", err)
		}

		startTime, startErr := time.Parse(time.RFC3339Nano, containerState.State.StartedAt)
		finishTime, finishErr := time.Parse(time.RFC3339Nano, containerState.State.FinishedAt)

		if startErr == nil && finishErr == nil && !startTime.IsZero() && !finishTime.IsZero() {
			timeUsage := finishTime.Sub(startTime)
			task.ExecutionTimeMs = int(timeUsage.Milliseconds())
		}

		// 非零退出码表示运行时错误
		if containerState.State.OOMKilled {
			task.Status = model.StatusMemoryLimitExceed
		} else if containerState.State.ExitCode != 0 {
			task.Status = model.StatusRuntimeError
		} else {
			task.Status = model.StatusCompleted
		}
	case err := <-errCh:
		return err
	case <-runCtx.Done():
		task.Status = model.StatusTimeLimitExceed
	}

	// Process execution results by reading output files
	stdoutPath := filepath.Join(tmpDir, "stdout.txt")
	stderrPath := filepath.Join(tmpDir, "stderr.txt")

	// Read program output
	if outData, err := os.ReadFile(stdoutPath); err == nil {
		task.Stdout = string(outData)
	}
	if outData, err := os.ReadFile(stderrPath); err == nil {
		task.Stderr = string(outData)
	}

	// 确保内存统计有效性（至少1MB）
	if maxMem > 0 {
		task.MemoryUsageKb = int(maxMem / 1024)
	} else {
		// 如果统计失败，尝试从OOM状态获取
		if task.Status == model.StatusMemoryLimitExceed {
			task.MemoryUsageKb = int(w.cfg.Limit.Memory * 1024) // 使用配置的内存限制值
		} else {
			task.MemoryUsageKb = -1 // 表示统计不可用
		}
	}

	return nil
}
