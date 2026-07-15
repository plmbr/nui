// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"fmt"
	"os"
	"strings"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/providers/anthropic"
	"github.com/mozilla-ai/any-llm-go/providers/gemini"
	"github.com/mozilla-ai/any-llm-go/providers/ollama"
	"github.com/mozilla-ai/any-llm-go/providers/openai"
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

// NewLLMProvider constructs an any-llm-go provider from harness ADL fields.
func NewLLMProvider(h model.ADLHarness, env map[string]string) (anyllm.Provider, error) {
	profile := ResolveAPIProviderProfile(h)
	apiKey, err := resolveAPIKey(profile, env)
	if err != nil {
		return nil, err
	}

	var opts []anyllm.Option
	if apiKey != "" {
		opts = append(opts, anyllm.WithAPIKey(apiKey))
	}
	if base := resolveAPIBaseURL(profile, h, env); base != "" {
		opts = append(opts, anyllm.WithBaseURL(base))
	}

	switch profile.ProviderID {
	case "anthropic":
		return anthropic.New(opts...)
	case "openai":
		return openai.New(opts...)
	case "gemini":
		return gemini.New(opts...)
	case "ollama":
		return ollama.New(opts...)
	default:
		return nil, fmt.Errorf("unsupported api provider %q", profile.ProviderID)
	}
}
