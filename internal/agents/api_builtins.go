// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"nui/internal/hitl"
	"nui/internal/model"
)

// APIBuiltinOrder is the stable preference order for default agent selection.
var APIBuiltinOrder = []string{
	"anthropic",
	"openai",
	"gemini",
	"openrouter",
	"ollama",
}

var apiBuiltinAgentDefs = []model.ADLDefinition{
	{
		ID:          "anthropic",
		Name:        "Claude API",
		Description: "Claude models via the Anthropic API",
		Tags:        []string{"builtin", "api"},
		Harness: model.ADLHarness{
			Type:     "api",
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
		},
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
		PromptSuggestions: builtinPromptSuggestions,
	},
	{
		ID:          "openai",
		Name:        "OpenAI",
		Description: "GPT models via the OpenAI API",
		Tags:        []string{"builtin", "api"},
		Harness: model.ADLHarness{
			Type:     "api",
			Provider: "openai",
			Model:    "gpt-4o-mini",
		},
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
		PromptSuggestions: builtinPromptSuggestions,
	},
	{
		ID:          "gemini",
		Name:        "Gemini",
		Description: "Google Gemini via the Gemini API",
		Tags:        []string{"builtin", "api"},
		Harness: model.ADLHarness{
			Type:     "api",
			Provider: "gemini",
			Model:    "gemini-3.5-flash",
		},
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
		PromptSuggestions: builtinPromptSuggestions,
	},
	{
		ID:          "openrouter",
		Name:        "OpenRouter",
		Description: "Multi-model routing via OpenRouter",
		Tags:        []string{"builtin", "api"},
		Harness: model.ADLHarness{
			Type:      "api",
			Provider:  "openrouter",
			Model:     "anthropic/claude-sonnet-4",
			BaseURL:   "https://openrouter.ai/api/v1",
			APIKeyEnv: "OPENROUTER_API_KEY",
		},
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
		PromptSuggestions: builtinPromptSuggestions,
	},
	{
		ID:          "ollama",
		Name:        "Ollama",
		Description: "Local models via Ollama",
		Tags:        []string{"builtin", "api"},
		Harness: model.ADLHarness{
			Type:     "api",
			Provider: "ollama",
		},
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
		PromptSuggestions: builtinPromptSuggestions,
	},
}
