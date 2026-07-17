// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"strings"

	"nui/internal/llm"
)

// ensureOllamaModel picks a model that exists on the local Ollama instance.
// When the requested model is missing, the first installed model is used.
func ensureOllamaModel(ctx context.Context, provider llm.Provider, requested string) (string, error) {
	lister, ok := provider.(llm.ModelLister)
	if !ok {
		return strings.TrimSpace(requested), nil
	}
	resp, err := lister.ListModels(ctx)
	if err != nil {
		if strings.TrimSpace(requested) != "" {
			return requested, nil
		}
		return "", fmt.Errorf("ollama: list models: %w", err)
	}
	ids := make([]string, 0, len(resp.Data))
	for _, m := range resp.Data {
		if id := strings.TrimSpace(m.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return pickOllamaModel(strings.TrimSpace(requested), ids)
}

func pickOllamaModel(requested string, installed []string) (string, error) {
	if len(installed) == 0 {
		return "", fmt.Errorf("ollama: no models installed (run `ollama pull <model>`)")
	}
	if requested != "" {
		for _, id := range installed {
			if id == requested || strings.HasPrefix(id, requested+":") {
				return id, nil
			}
		}
	}
	return installed[0], nil
}
