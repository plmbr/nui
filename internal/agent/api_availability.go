// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"strings"

	"nui/internal/model"
	"nui/internal/store"
)

const openRouterDefaultBaseURL = "https://openrouter.ai/api/v1"

// APIProviderProfile describes how to connect an api harness to an LLM provider.
type APIProviderProfile struct {
	ProviderID string
	APIKeyEnvs []string
	BaseURL    string
	BaseURLEnv string // env var for API base URL override (e.g. OPENAI_BASE_URL)
	NeedsKey   bool
}

// CredentialFieldSpec describes a user-editable credential env var for the Credentials UI.
type CredentialFieldSpec struct {
	Key         string
	Label       string
	Description string
	Group       string
	Secret      bool // true for API keys; false for host/base URL overrides
}

// CredentialFieldSpecs returns the curated list of credential env vars managed in Settings.
func CredentialFieldSpecs() []CredentialFieldSpec {
	return []CredentialFieldSpec{
		{
			Key:         "ANTHROPIC_API_KEY",
			Label:       "Anthropic API key",
			Description: "Used by the Claude API builtin, Claude Code, and Anthropic-compatible agents.",
			Group:       "Anthropic",
			Secret:      true,
		},
		{
			Key:         "ANTHROPIC_BASE_URL",
			Label:       "Anthropic base URL",
			Description: "Optional API base URL override.",
			Group:       "Anthropic",
			Secret:      false,
		},
		{
			Key:         "OPENAI_API_KEY",
			Label:       "OpenAI API key",
			Description: "Used by the OpenAI builtin.",
			Group:       "OpenAI",
			Secret:      true,
		},
		{
			Key:         "OPENAI_BASE_URL",
			Label:       "OpenAI base URL",
			Description: "Optional API base URL override.",
			Group:       "OpenAI",
			Secret:      false,
		},
		{
			Key:         "GEMINI_API_KEY",
			Label:       "Gemini API key",
			Description: "Used by the Gemini builtin. GOOGLE_API_KEY is also accepted from the environment.",
			Group:       "Gemini",
			Secret:      true,
		},
		{
			Key:         "OPENROUTER_API_KEY",
			Label:       "OpenRouter API key",
			Description: "Used by the OpenRouter builtin.",
			Group:       "OpenRouter",
			Secret:      true,
		},
		{
			Key:         "OLLAMA_HOST",
			Label:       "Ollama host",
			Description: "Optional Ollama base URL (default http://127.0.0.1:11434).",
			Group:       "Ollama",
			Secret:      false,
		},
	}
}

// IsManagedCredentialKey reports whether key is editable via the Credentials UI.
func IsManagedCredentialKey(key string) bool {
	key = strings.TrimSpace(key)
	for _, spec := range CredentialFieldSpecs() {
		if spec.Key == key {
			return true
		}
	}
	return false
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

// lookupCredentialValue resolves a credential key with priority:
// request/ADL env map → process environment → ~/.nui/secrets.json.
func lookupCredentialValue(key string, env map[string]string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if env != nil {
		if v := strings.TrimSpace(env[key]); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return store.SecretEnv(key)
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
		if lookupCredentialValue(envKey, h.Env) != "" {
			return true
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
