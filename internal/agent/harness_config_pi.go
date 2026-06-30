// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"loop/internal/model"
)

const piSystemPromptFile = "SYSTEM.md"

type piHarnessProvisioner struct{}

func (piHarnessProvisioner) provision(sessionConfigDir string, deps HarnessDeps) error {
	agentDir := piAgentConfigDir(sessionConfigDir)
	if err := os.MkdirAll(agentDir, 0700); err != nil {
		return err
	}
	if err := writePiSystemPrompt(agentDir, deps.SystemPrompt); err != nil {
		return err
	}
	if err := writePiMCPConfig(agentDir, deps.MCPServers); err != nil {
		return err
	}
	if err := installHarnessSkills("pi", sessionConfigDir, deps.WorkingDir, deps.Skills); err != nil {
		return fmt.Errorf("install skills: %w", err)
	}
	if _, err := installHarnessRules("pi", sessionConfigDir, deps.ResolvedRules); err != nil {
		return fmt.Errorf("install rules: %w", err)
	}
	return writeHarnessManifest(sessionConfigDir, "pi", deps, map[string]any{
		"agentDir":         agentDir,
		"systemPromptFile": piSystemPromptFile,
		"rulesDir":         filepath.Join(piAgentSubdir, "rules"),
		"configEnv":        envPiCodingAgentDir,
	})
}

func writePiSystemPrompt(agentDir, systemPrompt string) error {
	path := filepath.Join(agentDir, piSystemPromptFile)
	if strings.TrimSpace(systemPrompt) == "" {
		_ = os.Remove(path)
		return nil
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(systemPrompt)+"\n"), 0644)
}

func writePiMCPConfig(agentDir string, servers []model.ADLMCPServer) error {
	cfgPath := filepath.Join(agentDir, "mcp.json")
	if len(servers) == 0 {
		_ = os.Remove(cfgPath)
		return nil
	}

	mcpServers := make(map[string]map[string]any, len(servers))
	for _, srv := range servers {
		name := strings.TrimSpace(srv.Name)
		if name == "" {
			continue
		}
		entry, err := adlMCPServerToClaude(srv)
		if err != nil {
			return fmt.Errorf("mcp server %q: %w", name, err)
		}
		mcpServers[name] = entry
	}
	if len(mcpServers) == 0 {
		_ = os.Remove(cfgPath)
		return nil
	}

	data, err := json.MarshalIndent(map[string]any{"mcpServers": mcpServers}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}
