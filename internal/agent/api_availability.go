// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"strings"

	"loop/internal/model"
)

const openRouterDefaultBaseURL = "https://openrouter.ai/api/v1"

// APIProviderProfile describes how to connect an api harness to any-llm-go.
type APIProviderProfile struct {
	ProviderID string
	APIKeyEnvs []string
	BaseURL    string
	BaseURLEnv string // env var for API base URL override (e.g. OPENAI_BASE_URL)
	NeedsKey   bool
}

var defaultAPIProfiles = map[string]APIProviderProfile{
	"anthropic": {
		ProviderID: "anthropic",
		APIKeyEnvs: []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_OAUTH_TOKEN"},
		BaseURLEnv: "ANTHROPIC_BASE_URL",
		NeedsKey:   true,
	},
	"openai": {
		ProviderID: "openai",
		APIKeyEnvs: []string{"OPENAI_API_KEY"},
		BaseURLEnv: "OPENAI_BASE_URL",
		NeedsKey:   true,
	},
	"gemini": {
		ProviderID: "gemini",
		APIKeyEnvs: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
		NeedsKey:   true,
	},
	"openrouter": {
		ProviderID: "openai",
		APIKeyEnvs: []string{"OPENROUTER_API_KEY"},
		BaseURL:    openRouterDefaultBaseURL,
		NeedsKey:   true,
	},
	"ollama": {
		ProviderID: "ollama",
		BaseURLEnv: "OLLAMA_HOST",
		NeedsKey:   false,
	},
}

// ResolveAPIProviderProfile returns connection metadata for an api harness.
func ResolveAPIProviderProfile(h model.ADLHarness) APIProviderProfile {
	provider := strings.TrimSpace(h.Provider)
	if provider == "" {
		provider = "anthropic"
	}
	profile, ok := defaultAPIProfiles[provider]
	if !ok {
		profile = APIProviderProfile{ProviderID: provider, NeedsKey: true}
	}
	if base := strings.TrimSpace(h.BaseURL); base != "" {
		profile.BaseURL = base
	}
	if env := strings.TrimSpace(h.APIKeyEnv); env != "" {
		profile.APIKeyEnvs = []string{env}
	}
	return profile
}

// APIHarnessAvailable reports whether the api harness credentials are configured.
func APIHarnessAvailable(h model.ADLHarness) bool {
	profile := ResolveAPIProviderProfile(h)
	if !profile.NeedsKey {
		return true
	}
	for _, envKey := range profile.APIKeyEnvs {
		if strings.TrimSpace(os.Getenv(envKey)) != "" {
			return true
		}
	}
	if len(h.Env) > 0 {
		for _, envKey := range profile.APIKeyEnvs {
			if v := strings.TrimSpace(h.Env[envKey]); v != "" {
				return true
			}
		}
	}
	return false
}

// APIHarnessAvailableDef checks an ADL definition with harness.type api.
func APIHarnessAvailableDef(def model.ADLDefinition) bool {
	if def.Harness.Type != "api" {
		return false
	}
	return APIHarnessAvailable(def.Harness)
}
