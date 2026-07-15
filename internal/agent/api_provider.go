// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"fmt"
	"os"
	"strings"

	"loop/internal/llm"
	"loop/internal/model"
)

func resolveAPIKey(profile APIProviderProfile, env map[string]string) (string, error) {
	for _, key := range profile.APIKeyEnvs {
		if env != nil {
			if v := strings.TrimSpace(env[key]); v != "" {
				return v, nil
			}
		}
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v, nil
		}
	}
	if !profile.NeedsKey {
		return "", nil
	}
	if len(profile.APIKeyEnvs) == 0 {
		return "", fmt.Errorf("api key not configured")
	}
	return "", fmt.Errorf("%s not set", profile.APIKeyEnvs[0])
}

func resolveAPIBaseURL(profile APIProviderProfile, h model.ADLHarness, env map[string]string) string {
	if base := strings.TrimSpace(profile.BaseURL); base != "" {
		return base
	}
	if base := strings.TrimSpace(h.BaseURL); base != "" {
		return base
	}
	if profile.BaseURLEnv == "" {
		return ""
	}
	if env != nil {
		if v := strings.TrimSpace(env[profile.BaseURLEnv]); v != "" {
			return v
		}
	}
	return strings.TrimSpace(os.Getenv(profile.BaseURLEnv))
}

// NewLLMProvider constructs an LLM provider from harness ADL fields.
func NewLLMProvider(h model.ADLHarness, env map[string]string) (llm.Provider, error) {
	profile := ResolveAPIProviderProfile(h)
	apiKey, err := resolveAPIKey(profile, env)
	if err != nil {
		return nil, err
	}
	base := resolveAPIBaseURL(profile, h, env)
	return llm.NewProvider(profile.ProviderID, apiKey, base)
}
