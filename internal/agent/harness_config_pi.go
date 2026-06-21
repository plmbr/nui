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
	if deps.Skill != "" {
		if err := installPiSkill(agentDir, deps.Skill); err != nil {
			return fmt.Errorf("install skill: %w", err)
		}
	}
	return writeHarnessManifest(sessionConfigDir, "pi", deps, map[string]any{
		"agentDir":         agentDir,
		"systemPromptFile": piSystemPromptFile,
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

func installPiSkill(agentDir, skillPath string) error {
	src, skillName, err := resolveSkillSource(skillPath)
	if err != nil {
		return err
	}
	destDir := filepath.Join(agentDir, "skills", skillName)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	return copyFile(filepath.Join(src, "SKILL.md"), filepath.Join(destDir, "SKILL.md"))
}
