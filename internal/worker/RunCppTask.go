package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"runbin/internal/config"
	"runbin/internal/model"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Usage struct {
	ExitStatus int64   `json:"exit_status"`
	MaxMemory  int64   `json:"max_memory"`
	RealTime   float64 `json:"real_time"`
}

func compileCpp(ctx context.Context, task *model.Paste, cli *client.Client, tmpDir string, cfg *config.WorkerConfig) error {
	hostConfig := &container.HostConfig{
		Binds: []string{tmpDir + ":/app"},
		Resources: container.Resources{
			Memory:   int64(cfg.Limit.Memory * 1024 * 1024),
			CPUQuota: int64(cfg.Limit.Cpu * 100000),
		},
		NetworkMode: "none",
	}

	// 写入main.cpp
	codePath := filepath.Join(tmpDir, "main.cpp")
	if err := os.WriteFile(codePath, []byte(task.Code), 0644); err != nil {
		return fmt.Errorf("write code file error: %v", err)
	}

	compliteCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Limit.Time)*time.Second)
	defer cancel()

	// 创建容器
	resp, err := cli.ContainerCreate(compliteCtx, &container.Config{
		Image: cfg.CompilerImage,
		Cmd:   []string{"sh", "-c", "g++ -std=c++20 /app/main.cpp -o /app/output > /app/compile.txt 2>&1"},
	}, hostConfig, nil, nil, filepath.Base(tmpDir)+"_builder")
	if err != nil {
		return fmt.Errorf("create compile container error: %v", err)
	}
	defer cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{
		Force: true,
	})

	// 启动容器
	if err := cli.ContainerStart(compliteCtx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start compile container: %v", err)
	}

	// 等待
	statusCh, errCh := cli.ContainerWait(compliteCtx, resp.ID, container.WaitConditionNotRunning)
	select {
	case status := <-statusCh:

		// Read compilation log
		if logData, err := os.ReadFile(filepath.Join(tmpDir, "compile.txt")); err == nil {
			task.CompileLog = string(logData)
		}

		// Non-zero exit code indicates compilation failure
		if status.StatusCode != 0 {
			task.Status = model.StatusCompileError
		}
		return nil
	case err := <-errCh:
		return err
	case <-compliteCtx.Done():
		task.Status = model.StatusCompileError
		task.CompileLog = "Compile process exceeded time limit"
		return nil
	}
}

func runCpp(ctx context.Context, task *model.Paste, cli *client.Client, tmpDir string, cfg *config.WorkerConfig) error {
	hostConfig := &container.HostConfig{
		Binds: []string{tmpDir + ":/app"},
		Resources: container.Resources{
			Memory:   int64(cfg.Limit.Memory * 1024 * 1024),
			CPUQuota: int64(cfg.Limit.Cpu * 100000),
		},
		NetworkMode: "none",
	}

	// 写入 input.txt
	inputPath := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(inputPath, []byte(task.Stdin), 0644); err != nil {
		return fmt.Errorf("write input file error: %v", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Limit.Time)*time.Second)
	defer cancel()

	// Create runner container configuration
	resp, err := cli.ContainerCreate(runCtx, &container.Config{
		Image: cfg.CompilerImage,
		Cmd:   []string{"sh", "-c", `/usr/bin/time --format='{"exit_status":%x,"max_memory":%M,"real_time":%e}' -o /app/usage.json /app/output < /app/input.txt > /app/stdout.txt 2> /app/stderr.txt`},
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

	// 等待容器完成
	statusCh, errCh := cli.ContainerWait(runCtx, resp.ID, container.WaitConditionNotRunning)

	// 处理执行结果
	select {
	case status := <-statusCh:

		// 非零退出码表示运行时错误
		if status.StatusCode == 137 {
			task.Status = model.StatusMemoryLimitExceed
		} else if status.StatusCode != 0 {
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
	usagePath := filepath.Join(tmpDir, "usage.json")

	// Read program output
	if outData, err := os.ReadFile(stdoutPath); err == nil {
		task.Stdout = string(outData)
	}
	if outData, err := os.ReadFile(stderrPath); err == nil {
		task.Stderr = string(outData)
	}
	if usageData, err := os.ReadFile(usagePath); err == nil {
		var usage Usage
		json.Unmarshal(usageData, &usage)
		task.MemoryUsageKb = int(usage.MaxMemory)
		task.ExecutionTimeMs = int(usage.RealTime * 1000) 
	}
	return nil
}

func (w *Worker) RunCppTask(ctx context.Context, task *model.Paste, cli *client.Client) error {
	// 临时文件夹
	tmpDir, err := os.MkdirTemp("", "cpp_compile_")
	if err != nil {
		return fmt.Errorf("create temp dir error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := compileCpp(ctx, task, cli, tmpDir, w.cfg); err != nil {
		return err
	}

	if task.Status == model.StatusCompileError {
		return nil
	}

	return runCpp(ctx, task, cli, tmpDir, w.cfg)
}
