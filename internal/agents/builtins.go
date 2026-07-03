// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import "loop/internal/model"

var builtinAgentDefs = []model.ADLDefinition{
	{
		ID:          "claude-code",
		Name:        "Claude Code",
		Description: "Claude Code running as a local subprocess",
		Harness:     model.ADLHarness{Type: "claude-code", Sandbox: "none"},
	},
	{
		ID:          "pi",
		Name:        "Pi",
		Description: "Pi running as a local subprocess",
		Harness:     model.ADLHarness{Type: "pi", Sandbox: "none"},
	},
	{
		ID:          "codex",
		Name:        "Codex",
		Description: "Codex running as a local subprocess",
		Harness:     model.ADLHarness{Type: "codex", Sandbox: "none"},
	},
	{
		ID:          "opencode",
		Name:        "OpenCode",
		Description: "OpenCode running as a local subprocess",
		Harness:     model.ADLHarness{Type: "opencode", Sandbox: "none"},
	},
}
