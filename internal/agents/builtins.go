// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"nui/internal/hitl"
	"nui/internal/model"
)

var builtinPromptSuggestions = []model.ADLPromptSuggestion{
	{
		Title:  "Surprise me",
		Icon:   "sparkles",
		Prompt: "Surprise me with something fun you can do right now. Keep it short and cheerful.",
	},
	{
		Title:  "Hot take",
		Icon:   "flame",
		Prompt: "Share a playful hot take about anything, then argue the opposite side just as passionately.",
	},
	{
		Title:  "Play a game",
		Icon:   "gamepad-2",
		Prompt: "Invent a silly chat game we can play together. Explain the rules in one paragraph.",
	},
}

// builtinAgentDefs are the compiled-in ADL definitions shipped with nui.
var builtinAgentDefs = []model.ADLDefinition{
	{
		ID:                "claude-code",
		Name:              "Claude Code",
		Description:       "Claude Code running as a local subprocess",
		Tags:              []string{"builtin", "cli"},
		Harness:           model.ADLHarness{Type: "claude-code", Sandbox: "none", Permissions: hitl.PermissionsBypass},
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
		WorkingDirInput:   true,
		PromptSuggestions: builtinPromptSuggestions,
	},
	{
		ID:                "pi",
		Name:              "Pi",
		Description:       "Pi running as a local subprocess",
		Tags:              []string{"builtin", "cli"},
		Harness:           model.ADLHarness{Type: "pi", Sandbox: "none"},
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
		WorkingDirInput:   true,
		PromptSuggestions: builtinPromptSuggestions,
	},
	{
		ID:                "codex",
		Name:              "Codex",
		Description:       "Codex running as a local subprocess",
		Tags:              []string{"builtin", "cli"},
		Harness:           model.ADLHarness{Type: "codex", Sandbox: "none", Permissions: hitl.PermissionsBypass},
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
		WorkingDirInput:   true,
		PromptSuggestions: builtinPromptSuggestions,
	},
	{
		ID:                "opencode",
		Name:              "OpenCode",
		Description:       "OpenCode running as a local subprocess",
		Tags:              []string{"builtin", "cli"},
		Harness:           model.ADLHarness{Type: "opencode", Sandbox: "none"},
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
		WorkingDirInput:   true,
		PromptSuggestions: builtinPromptSuggestions,
	},
}

// BuiltinAgentDefs returns compiled-in ADL agent definitions shipped with nui.
func BuiltinAgentDefs() []model.ADLDefinition {
	out := make([]model.ADLDefinition, 0, len(builtinAgentDefs)+len(apiBuiltinAgentDefs))
	out = append(out, builtinAgentDefs...)
	out = append(out, apiBuiltinAgentDefs...)
	return out
}
