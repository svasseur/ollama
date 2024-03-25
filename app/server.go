package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ollama/ollama/api"
)

func SpawnServer(ctx context.Context, command string) (chan int, error) {
	done := make(chan int)

	logDir := filepath.Dir(ServerLogFile)
	_, err := os.Stat(logDir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return done, fmt.Errorf("create ollama server log dir %s: %v", logDir, err)
		}
	}

	cmd := getCmd(ctx, command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return done, fmt.Errorf("failed to spawn server stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return done, fmt.Errorf("failed to spawn server stderr pipe: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return done, fmt.Errorf("failed to spawn server stdin pipe: %w", err)
	}

	// TODO - rotation
	logFile, err := os.OpenFile(ServerLogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return done, fmt.Errorf("failed to create server log: %w", err)
	}
	go func() {
		defer logFile.Close()
		io.Copy(logFile, stdout) //nolint:errcheck
	}()
	go func() {
		defer logFile.Close()
		io.Copy(logFile, stderr) //nolint:errcheck
	}()

	// run the command and wait for it to finish
	if err := cmd.Start(); err != nil {
		return done, fmt.Errorf("failed to start server %w", err)
	}
	if cmd.Process != nil {
		slog.Info(fmt.Sprintf("started ollama server with pid %d", cmd.Process.Pid))
	}
	slog.Info(fmt.Sprintf("ollama server logs %s", ServerLogFile))

	go func() {
		// Keep the server running unless we're shuttind down the app
		crashCount := 0
		for {
			cmd.Wait() //nolint:errcheck
			stdin.Close()
			var code int
			if cmd.ProcessState != nil {
				code = cmd.ProcessState.ExitCode()
			}

			select {
			case <-ctx.Done():
				slog.Debug(fmt.Sprintf("server shutdown with exit code %d", code))
				done <- code
				return
			default:
				crashCount++
				slog.Warn(fmt.Sprintf("server crash %d - exit code %d - respawning", crashCount, code))
				time.Sleep(500 * time.Millisecond)
				if err := cmd.Start(); err != nil {
					slog.Error(fmt.Sprintf("failed to restart server %s", err))
					// Keep trying, but back off if we keep failing
					time.Sleep(time.Duration(crashCount) * time.Second)
				}
			}
		}
	}()
	return done, nil
}

func isServerRunning(ctx context.Context) bool {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		slog.Info("unable to connect to server")
		return false
	}
	err = client.Heartbeat(ctx)
	if err != nil {
		slog.Debug(fmt.Sprintf("heartbeat from server: %s", err))
		slog.Info("unable to connect to server")
		return false
	}
	return true
}
