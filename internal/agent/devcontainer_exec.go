// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"nui/internal/devcontainer"
)

const sandboxDevcontainer = "devcontainer"

func devcontainerExecCommand(ctx context.Context, workspaceFolder, bin string, args []string) *exec.Cmd {
	workspaceFolder = strings.TrimSpace(workspaceFolder)
	execArgs := []string{"exec", "--workspace-folder", workspaceFolder, bin}
	execArgs = append(execArgs, args...)
	return exec.CommandContext(ctx, "devcontainer", execArgs...)
}

func useDevcontainerSandbox(sandbox, workspace string) bool {
	return normalizeSandbox(sandbox) == sandboxDevcontainer && strings.TrimSpace(workspace) != ""
}

func requireDevcontainer(sandbox, workspace string) error {
	if !useDevcontainerSandbox(sandbox, workspace) {
		return nil
	}
	if !devcontainer.Available() {
		return fmt.Errorf("devcontainer sandbox requested but devcontainer CLI not found on PATH")
	}
	return nil
}

func dockerExecCommand(ctx context.Context, containerID, bin string, args []string) *exec.Cmd {
	execArgs := []string{"exec", containerID, bin}
	execArgs = append(execArgs, args...)
	return exec.CommandContext(ctx, "docker", execArgs...)
}
