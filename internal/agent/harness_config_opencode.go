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

const opencodeConfigFile = "opencode.json"
const opencodeInstructionsFile = "INSTRUCTIONS.md"

type opencodeHarnessProvisioner struct{}

func (opencodeHarnessProvisioner) provision(configDir string, deps HarnessDeps) error {
	if err := writeOpenCodeInstructions(configDir, deps.SystemPrompt); err != nil {
		return err
	}
	if err := writeOpenCodeConfig(configDir, deps); err != nil {
		return err
	}
	if err := installHarnessSkills("opencode", configDir, deps.WorkingDir, deps.Skills); err != nil {
		return fmt.Errorf("install skills: %w", err)
	}
	return writeHarnessManifest(configDir, "opencode", deps, map[string]any{
		"configFile":       opencodeConfigFile,
		"instructionsFile": opencodeInstructionsFile,
		"configEnv":        envOpenCodeConfigDir,
	})
}

func writeOpenCodeInstructions(configDir, systemPrompt string) error {
	path := filepath.Join(configDir, opencodeInstructionsFile)
	if strings.TrimSpace(systemPrompt) == "" {
		_ = os.Remove(path)
		return nil
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(systemPrompt)+"\n"), 0644)
}

func writeOpenCodeConfig(configDir string, deps HarnessDeps) error {
	cfg := map[string]any{
		"$schema": "https://opencode.ai/config.json",
	}

	if strings.TrimSpace(deps.SystemPrompt) != "" {
		cfg["instructions"] = []string{"./" + opencodeInstructionsFile}
	}

	if len(deps.MCPServers) > 0 {
		mcp := make(map[string]map[string]any, len(deps.MCPServers))
		for _, srv := range deps.MCPServers {
			name := strings.TrimSpace(srv.Name)
			if name == "" {
				continue
			}
			entry, err := adlMCPServerToOpenCode(srv)
			if err != nil {
				return fmt.Errorf("mcp server %q: %w", name, err)
			}
			mcp[name] = entry
		}
		if len(mcp) > 0 {
			cfg["mcp"] = mcp
		}
	}

	if len(cfg) == 1 { // only $schema
		cfgPath := filepath.Join(configDir, opencodeConfigFile)
		_ = os.Remove(cfgPath)
		return nil
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, opencodeConfigFile), data, 0644)
}

func adlMCPServerToOpenCode(srv model.ADLMCPServer) (map[string]any, error) {
	entry := map[string]any{"enabled": true}

	if cmd := strings.TrimSpace(srv.Command); cmd != "" {
		command := []string{cmd}
		if len(srv.Args) > 0 {
			command = append(command, srv.Args...)
		}
		entry["type"] = "local"
		entry["command"] = command
		return entry, nil
	}

	url := strings.TrimSpace(srv.URL)
	if url == "" {
		return nil, fmt.Errorf("requires url or command")
	}
	entry["type"] = "remote"
	entry["url"] = url
	return entry, nil
}
