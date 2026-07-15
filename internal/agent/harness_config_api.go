// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"loop/internal/skills"
)

const apiSystemPromptFile = "LOOP_API_SYSTEM.md"

type apiHarnessProvisioner struct{}

func (apiHarnessProvisioner) provision(configDir string, deps HarnessDeps) error {
	prompt := assembleAPISystemPrompt(deps)
	if err := writeAPISystemPrompt(configDir, prompt); err != nil {
		return err
	}
	return writeHarnessManifest(configDir, "api", deps, map[string]any{
		"systemPromptFile": apiSystemPromptFile,
	})
}

func assembleAPISystemPrompt(deps HarnessDeps) string {
	var b strings.Builder
	if sp := strings.TrimSpace(deps.SystemPrompt); sp != "" {
		b.WriteString(sp)
		b.WriteString("\n\n")
	}
	for _, rule := range deps.ResolvedRules {
		name := strings.TrimSpace(rule.Name)
		body := strings.TrimSpace(rule.Content)
		if body == "" {
			continue
		}
		if name != "" {
			fmt.Fprintf(&b, "## Rule: %s\n\n", name)
		}
		b.WriteString(body)
		b.WriteString("\n\n")
	}
	if appendix := skills.PromptAppendix(skills.Context{WorkingDir: deps.WorkingDir}, deps.Skills); appendix != "" {
		b.WriteString(appendix)
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

func writeAPISystemPrompt(configDir, prompt string) error {
	path := filepath.Join(configDir, apiSystemPromptFile)
	if strings.TrimSpace(prompt) == "" {
		_ = os.Remove(path)
		return nil
	}
	return os.WriteFile(path, []byte(prompt+"\n"), 0644)
}

// APISystemPromptFromDeps returns the composite system prompt for an api harness run.
func APISystemPromptFromDeps(deps HarnessDeps) string {
	return assembleAPISystemPrompt(deps)
}
