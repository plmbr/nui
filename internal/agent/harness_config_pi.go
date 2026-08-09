// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nui/internal/model"
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
	if deps.seedsUserConfig() {
		if err := linkPiConfigFromUser(agentDir); err != nil {
			return fmt.Errorf("link user config: %w", err)
		}
	}
	return writeHarnessManifest(sessionConfigDir, "pi", deps, map[string]any{
		"agentDir":         agentDir,
		"systemPromptFile": piSystemPromptFile,
		"rulesDir":         filepath.Join(piAgentSubdir, "rules"),
		"configEnv":        envPiCodingAgentDir,
	})
}

// piUserConfigEntries are user-level Pi files that carry credentials, provider endpoints,
// model catalogs, and installed packages. Without them an isolated PI_CODING_AGENT_DIR has
// no configured provider and Pi rejects every prompt with "No API key found".
var piUserConfigEntries = []string{
	"auth.json",
	"settings.json",
	"models.json",
	"keybindings.json",
	"npm",
	"bin",
	"sessions",
}

// linkPiConfigFromUser seeds an isolated Pi agent dir from ~/.pi/agent. The sessions
// directory is linked rather than copied because store.LoadPiHistory reads transcripts
// from the user-level path.
func linkPiConfigFromUser(agentDir string) error {
	srcDir, err := userPiAgentDir()
	if err != nil {
		return err
	}
	same, err := sameDir(srcDir, agentDir)
	if err != nil || same {
		return err
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "sessions"), 0700); err != nil {
		return err
	}

	names := append([]string(nil), piUserConfigEntries...)
	matches, err := filepath.Glob(filepath.Join(srcDir, "models-*.json"))
	if err != nil {
		return err
	}
	for _, match := range matches {
		names = append(names, filepath.Base(match))
	}
	return linkUserConfigEntries(srcDir, agentDir, names)
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
