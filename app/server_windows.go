package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

func getCLIFullPath(command string) string {
	cmdPath := ""
	appExe, err := os.Executable()
	if err == nil {
		cmdPath = filepath.Join(filepath.Dir(appExe), command)
		_, err := os.Stat(cmdPath)
		if err == nil {
			return cmdPath
		}
	}
	cmdPath, err = exec.LookPath(command)
	if err == nil {
		_, err := os.Stat(cmdPath)
		if err == nil {
			return cmdPath
		}
	}
	pwd, err := os.Getwd()
	if err == nil {
		cmdPath = filepath.Join(pwd, command)
		_, err = os.Stat(cmdPath)
		if err == nil {
			return cmdPath
		}
	}

	return command
}

func getCmd(ctx context.Context, exePath string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, exePath, "serve")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	}
	return cmd
}
