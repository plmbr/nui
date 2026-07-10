// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"loop/internal/model"
)

func normalizeSandbox(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func sandboxFromConfig(config map[string]any) string {
	if config == nil {
		return "none"
	}
	s, _ := config["sandbox"].(string)
	return normalizeSandbox(s)
}

func harnessBuiltinConfig(harness model.ADLHarness) map[string]any {
	return map[string]any{"sandbox": harness.Sandbox}
}

func builtinAgentSandbox(ag Agent) string {
	switch a := ag.(type) {
	case *ClaudeCodeAgent:
		return normalizeSandbox(a.Sandbox)
	case *PiAgent:
		return normalizeSandbox(a.Sandbox)
	case *CodexAgent:
		return normalizeSandbox(a.Sandbox)
	case *OpenCodeAgent:
		return normalizeSandbox(a.Sandbox)
	default:
		return "none"
	}
}

func applyBuiltinSandbox(ag Agent, sandbox string) {
	sandbox = normalizeSandbox(sandbox)
	switch a := ag.(type) {
	case *ClaudeCodeAgent:
		if normalizeSandbox(a.Sandbox) != sandbox {
			a.Stop()
			a.Sandbox = sandbox
		}
	case *PiAgent:
		if normalizeSandbox(a.Sandbox) != sandbox {
			a.Stop()
			a.Sandbox = sandbox
		}
	case *CodexAgent:
		if normalizeSandbox(a.Sandbox) != sandbox {
			a.Stop()
			a.Sandbox = sandbox
		}
	case *OpenCodeAgent:
		if normalizeSandbox(a.Sandbox) != sandbox {
			a.Stop()
			a.Sandbox = sandbox
		}
	}
}

func applyDevcontainerWorkspace(ag Agent, workspace string) {
	applyDevcontainerRuntime(ag, workspace, "")
}

func applyDevcontainerRuntime(ag Agent, workspace, containerID string) {
	switch a := ag.(type) {
	case *ClaudeCodeAgent:
		a.DevcontainerWorkspace = workspace
		a.DevcontainerContainerID = containerID
	case *PiAgent:
		a.DevcontainerWorkspace = workspace
		a.DevcontainerContainerID = containerID
	case *CodexAgent:
		a.DevcontainerWorkspace = workspace
		a.DevcontainerContainerID = containerID
	case *OpenCodeAgent:
		a.DevcontainerWorkspace = workspace
		a.DevcontainerContainerID = containerID
	}
}

func devcontainerContainerIDFromConfig(config map[string]any) string {
	if config == nil {
		return ""
	}
	s, _ := config["devcontainerContainerID"].(string)
	return strings.TrimSpace(s)
}

func devcontainerWorkspaceFromConfig(config map[string]any) string {
	if config == nil {
		return ""
	}
	s, _ := config["devcontainerWorkspace"].(string)
	return strings.TrimSpace(s)
}
