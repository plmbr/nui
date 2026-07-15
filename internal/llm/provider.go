// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import "fmt"

// NewProvider constructs an LLM provider for the given provider id.
func NewProvider(providerID, apiKey, baseURL string) (Provider, error) {
	switch providerID {
	case "openai":
		return newOpenAIProvider("openai", apiKey, baseURL)
	case "anthropic":
		return newAnthropicProvider(apiKey, baseURL)
	case "gemini":
		return newGeminiProvider(apiKey, baseURL)
	case "ollama":
		return newOllamaProvider(baseURL)
	default:
		return nil, fmt.Errorf("unsupported api provider %q", providerID)
	}
}
