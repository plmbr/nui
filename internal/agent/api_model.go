// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"strings"

	"loop/internal/model"
)

// providerModelEnvKeys lists env vars that override the builtin default model (first match wins).
var providerModelEnvKeys = map[string][]string{
	"anthropic": {
		"ANTHROPIC_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"CLAUDE_MODEL",
	},
	"openai":     {"OPENAI_MODEL"},
	"gemini":     {"GEMINI_MODEL", "GOOGLE_MODEL"},
	"openrouter": {"OPENROUTER_MODEL"},
	"ollama":     {"OLLAMA_MODEL"},
}

// resolveAPIModel picks the model id for an api harness run.
// Priority: session agentConfig.model → provider env vars → harness ADL model.
func resolveAPIModel(req RunRequest, harness model.ADLHarness) string {
	if req.AgentConfig != nil {
		if m, ok := req.AgentConfig["model"].(string); ok {
			if m = strings.TrimSpace(m); m != "" {
				return m
			}
		}
	}
	provider := strings.TrimSpace(harness.Provider)
	for _, key := range providerModelEnvKeys[provider] {
		if req.Env != nil {
			if v := strings.TrimSpace(req.Env[key]); v != "" {
				return v
			}
		}
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	resolved := strings.TrimSpace(req.Model)
	if resolved == "" {
		resolved = strings.TrimSpace(harness.Model)
	}
	if provider == "anthropic" {
		if candidates := resolveAnthropicModelCandidates(req, harness, resolved); len(candidates) > 0 {
			return candidates[0]
		}
	}
	return resolved
}
