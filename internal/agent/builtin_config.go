// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "loop/internal/model"

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
