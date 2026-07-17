// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"strings"

	"nui/internal/model"
)

const anthropicBuiltinDefaultModel = "claude-sonnet-4-20250514"

// anthropicGatewayModelFallbacks are tried in order when ANTHROPIC_BASE_URL points
// at a non-standard endpoint and no explicit model is configured.
var anthropicGatewayModelFallbacks = []string{
	"claude-sonnet-4-6",
	"claude-3-5-sonnet-latest",
	"claude-3-5-sonnet-20241022",
	"sonnet",
}

func isCustomAnthropicEndpoint(h model.ADLHarness, env map[string]string) bool {
	profile := ResolveAPIProviderProfile(h)
	if profile.ProviderID != "anthropic" {
		return false
	}
	base := strings.ToLower(strings.TrimRight(strings.TrimSpace(resolveAPIBaseURL(profile, h, env)), "/"))
	if base == "" {
		return false
	}
	return !strings.Contains(base, "api.anthropic.com")
}

func anthropicExplicitModel(req RunRequest) string {
	if req.AgentConfig != nil {
		if m, ok := req.AgentConfig["model"].(string); ok {
			if m = strings.TrimSpace(m); m != "" {
				return m
			}
		}
	}
	for _, key := range providerModelEnvKeys["anthropic"] {
		if req.Env != nil {
			if v := strings.TrimSpace(req.Env[key]); v != "" {
				return v
			}
		}
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

// resolveAnthropicModelCandidates returns models to try for an anthropic api harness run.
// When ANTHROPIC_BASE_URL targets a custom gateway and no model is explicitly configured,
// gateway-friendly aliases are tried before the dated public-api default.
func resolveAnthropicModelCandidates(req RunRequest, harness model.ADLHarness, resolved string) []string {
	if explicit := anthropicExplicitModel(req); explicit != "" {
		return []string{explicit}
	}
	if !isCustomAnthropicEndpoint(harness, req.Env) {
		m := strings.TrimSpace(resolved)
		if m == "" {
			m = strings.TrimSpace(harness.Model)
		}
		if m == "" {
			return nil
		}
		return []string{m}
	}

	seen := map[string]bool{}
	var out []string
	add := func(m string) {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			return
		}
		seen[m] = true
		out = append(out, m)
	}
	for _, fb := range anthropicGatewayModelFallbacks {
		add(fb)
	}
	add(resolved)
	add(harness.Model)
	return out
}
