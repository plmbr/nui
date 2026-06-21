// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"loop/internal/model"
)

const claudeSystemPromptFile = "CLAUDE.md"

type claudeHarnessProvisioner struct{}

func (claudeHarnessProvisioner) provision(configDir string, deps HarnessDeps) error {
	if err := writeClaudeSystemPrompt(configDir, deps.SystemPrompt); err != nil {
		return err
	}
	if err := writeClaudeMCPConfig(configDir, deps.MCPServers); err != nil {
		return err
	}
	if deps.Skill != "" {
		if err := installClaudeSkill(configDir, deps.Skill); err != nil {
			return fmt.Errorf("install skill: %w", err)
		}
	}
	return writeHarnessManifest(configDir, "claude-code", deps, map[string]any{
		"systemPromptFile": claudeSystemPromptFile,
		"configEnv":        envClaudeConfigDir,
	})
}

func writeClaudeSystemPrompt(configDir, systemPrompt string) error {
	path := filepath.Join(configDir, claudeSystemPromptFile)
	if strings.TrimSpace(systemPrompt) == "" {
		_ = os.Remove(path)
		return nil
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(systemPrompt)+"\n"), 0644)
}

func writeClaudeMCPConfig(configDir string, servers []model.ADLMCPServer) error {
	cfgPath := filepath.Join(configDir, ".claude.json")
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

func adlMCPServerToClaude(srv model.ADLMCPServer) (map[string]any, error) {
	if cmd := strings.TrimSpace(srv.Command); cmd != "" {
		entry := map[string]any{
			"command": cmd,
		}
		if len(srv.Args) > 0 {
			entry["args"] = srv.Args
		}
		if t := strings.TrimSpace(srv.Type); t != "" {
			entry["type"] = t
		}
		return entry, nil
	}

	url := strings.TrimSpace(srv.URL)
	if url == "" {
		return nil, fmt.Errorf("requires url or command")
	}
	entry := map[string]any{"url": url}
	if t := strings.TrimSpace(srv.Type); t != "" {
		entry["type"] = t
	} else {
		entry["type"] = "http"
	}
	return entry, nil
}

func installClaudeSkill(configDir, skillPath string) error {
	src, skillName, err := resolveSkillSource(skillPath)
	if err != nil {
		return err
	}
	destDir := filepath.Join(configDir, ".claude", "skills", skillName)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	return copyFile(filepath.Join(src, "SKILL.md"), filepath.Join(destDir, "SKILL.md"))
}

func resolveSkillSource(skillPath string) (skillDir string, skillName string, err error) {
	src, err := expandPath(skillPath)
	if err != nil {
		return "", "", err
	}

	switch {
	case strings.HasSuffix(src, "SKILL.md"):
		skillDir = filepath.Dir(src)
		skillName = filepath.Base(skillDir)
	case isDir(src):
		skillDir = src
		skillName = filepath.Base(src)
	default:
		return "", "", fmt.Errorf("skill path %q is not a directory or SKILL.md file", skillPath)
	}

	skillFile := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		return "", "", fmt.Errorf("skill %q: missing SKILL.md", skillPath)
	}
	return skillDir, skillName, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
