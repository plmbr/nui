// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"nui/internal/model"
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

// antigravityModelEnvKeys override the Antigravity CLI harness model (first match wins).
var antigravityModelEnvKeys = []string{
	"ANTIGRAVITY_MODEL",
	"GEMINI_MODEL",
	"GOOGLE_MODEL",
}

// resolveAPIModel picks the model id for an api harness run.
// Priority: session agentConfig.model → provider env/secrets → harness ADL model.
func resolveAPIModel(req RunRequest, harness model.ADLHarness) string {
	if m := modelFromAgentConfig(req); m != "" {
		return m
	}
	provider := strings.TrimSpace(harness.Provider)
	if m := firstModelEnv(req.Env, providerModelEnvKeys[provider]...); m != "" {
		return m
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

// resolveAntigravityModel picks the model slug for the Antigravity CLI harness.
// Priority: session agentConfig.model → ANTIGRAVITY_MODEL / GEMINI_MODEL / GOOGLE_MODEL
// (ADL env, process env, or ~/.nui/secrets.json) → harness ADL model.
func resolveAntigravityModel(req RunRequest, harness model.ADLHarness) string {
	if m := modelFromAgentConfig(req); m != "" {
		return m
	}
	if m := firstModelEnv(req.Env, antigravityModelEnvKeys...); m != "" {
		return m
	}
	resolved := strings.TrimSpace(req.Model)
	if resolved == "" {
		resolved = strings.TrimSpace(harness.Model)
	}
	return resolved
}

func modelFromAgentConfig(req RunRequest) string {
	if req.AgentConfig == nil {
		return ""
	}
	m, _ := req.AgentConfig["model"].(string)
	return strings.TrimSpace(m)
}

func firstModelEnv(adlEnv map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := lookupCredentialValue(key, adlEnv); v != "" {
			return v
		}
	}
	return ""
}
