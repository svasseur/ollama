package main

import (
	"context"
	"os/exec"
)

func getCmd(ctx context.Context, exePath string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, exePath, "serve")
	return cmd
}
