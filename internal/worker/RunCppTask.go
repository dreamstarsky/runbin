package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"runbin/internal/model"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func (w *Worker) RunCppTask(ctx context.Context, task *model.Paste) error {
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
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %v", err)
	}

	hostConfig := &container.HostConfig{
		Binds: []string{tmpDir + ":/app"},
		Resources: container.Resources{
			Memory:   int64(w.cfg.Limit.Memory * 1024 * 1024),
			CPUQuota: int64(w.cfg.Limit.Cpu * 100000),
		},
		AutoRemove:  true,
		NetworkMode: "none",
	}

	// Create compile container configuration
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "gcc:14",
		Cmd:   []string{"sh", "-c", "g++ -std=c++20 /app/main.cpp -o /app/output > /app/compile.txt 2>&1"},
	}, hostConfig, nil, nil, filepath.Base(tmpDir)+"_builder")
	if err != nil {
		return fmt.Errorf("create compile container error: %v", err)
	}

	// Start compile container execution
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start compile container: %v", err)
	}

	// Wait for compile container completion
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	// Handle compile container execution results
	select {
	case <-statusCh:
		// Get compile container detailed state
		containerState, err := cli.ContainerInspect(ctx, resp.ID)
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
	case <-ctx.Done():
		task.Status = model.StatusCompileError
		task.CompileLog = "Compile process exceeded time limit"
		return nil
	}

	// Write input to input.txt
	inputPath := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(inputPath, []byte(task.Stdin), 0644); err != nil {
		return fmt.Errorf("write input file error: %v", err)
	}

	// Create runner container configuration
	resp, err = cli.ContainerCreate(ctx, &container.Config{
		Image: "gcc:14",
		Cmd:   []string{"sh", "-c", "sh -c \"/app/output < /app/input.txt > /app/stdout.txt\" > /app/stderr.txt 2>&1"},
	}, hostConfig, nil, nil, filepath.Base(tmpDir)+"_runner")
	if err != nil {
		return fmt.Errorf("create runner container error: %v", err)
	}

	// Start runner container execution
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start runner container: %v", err)
	}

	// Wait for container completion
	statusCh, errCh = cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	// Handle container execution results
	select {
	case <-statusCh:
		// Get container detailed state
		containerState, err := cli.ContainerInspect(ctx, resp.ID)
		if err != nil {
			return fmt.Errorf("container state inspection error: %v", err)
		}

		// Non-zero exit code indicates compilation failure
		if containerState.State.ExitCode != 0 {
			task.Status = model.StatusRuntimeError
		} else {
			task.Status = model.StatusCompleted
		}
	case err := <-errCh:
		return err
	case <-ctx.Done():
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

	return nil
}
