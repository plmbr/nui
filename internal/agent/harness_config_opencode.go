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

const opencodeConfigFile = "opencode.json"
const opencodeInstructionsFile = "INSTRUCTIONS.md"

type opencodeHarnessProvisioner struct{}

func (opencodeHarnessProvisioner) provision(configDir string, deps HarnessDeps) error {
	if err := writeOpenCodeInstructions(configDir, deps.SystemPrompt); err != nil {
		return err
	}
	rulePaths, err := installHarnessRules("opencode", configDir, deps.ResolvedRules)
	if err != nil {
		return fmt.Errorf("install rules: %w", err)
	}
	if err := writeOpenCodeConfig(configDir, deps, rulePaths); err != nil {
		return err
	}
	if err := installHarnessSkills("opencode", configDir, deps.WorkingDir, deps.Skills); err != nil {
		return fmt.Errorf("install skills: %w", err)
	}
	if deps.seedsUserConfig() {
		if err := linkOpenCodeConfigFromUser(configDir); err != nil {
			return fmt.Errorf("link user config: %w", err)
		}
	}
	return writeHarnessManifest(configDir, "opencode", deps, map[string]any{
		"configFile":       opencodeConfigFile,
		"instructionsFile": opencodeInstructionsFile,
		"rulesDir":         "rules",
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

// opencodeUserConfigEntries are user-level OpenCode config entries that nui does not
// generate. OPENCODE_CONFIG_DIR points at the session directory, so plugins and custom
// agents would otherwise disappear for the session.
var opencodeUserConfigEntries = []string{
	"auth.json",
	"package.json",
	"node_modules",
	"plugin",
	"agent",
	"command",
	"themes",
}

func linkOpenCodeConfigFromUser(configDir string) error {
	srcDir, err := userOpenCodeConfigDir()
	if err != nil {
		return err
	}
	return linkUserConfigEntries(srcDir, configDir, opencodeUserConfigEntries)
}

// userOpenCodeConfig loads the user's global opencode config so session config generated
// into an isolated OPENCODE_CONFIG_DIR keeps their provider and model settings. A config
// that fails to parse (for example JSONC with comments) is skipped.
func userOpenCodeConfig() map[string]any {
	srcDir, err := userOpenCodeConfigDir()
	if err != nil {
		return nil
	}
	for _, name := range []string{opencodeConfigFile, "opencode.jsonc"} {
		data, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			continue
		}
		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[opencode] ignoring unparsable user config %s: %v\n", name, err)
			continue
		}
		return cfg
	}
	return nil
}

func writeOpenCodeConfig(configDir string, deps HarnessDeps, rulePaths []string) error {
	generated := map[string]any{}

	var instructions []string
	if strings.TrimSpace(deps.SystemPrompt) != "" {
		instructions = append(instructions, "./"+opencodeInstructionsFile)
	}
	for _, p := range rulePaths {
		instructions = append(instructions, "./"+p)
	}
	if len(instructions) > 0 {
		generated["instructions"] = instructions
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
			generated["mcp"] = mcp
		}
	}

	cfg := userOpenCodeConfig()
	if len(cfg) == 0 {
		if len(generated) == 0 {
			_ = os.Remove(filepath.Join(configDir, opencodeConfigFile))
			return nil
		}
		cfg = map[string]any{}
	}
	// User instructions are relative to their own config dir and would not resolve here.
	delete(cfg, "instructions")
	for k, v := range generated {
		cfg[k] = v
	}
	cfg["$schema"] = "https://opencode.ai/config.json"

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
		if len(srv.Env) > 0 {
			entry["environment"] = srv.Env
		}
		return entry, nil
	}

	url := strings.TrimSpace(srv.URL)
	if url == "" {
		return nil, fmt.Errorf("requires url or command")
	}
	entry["type"] = "remote"
	entry["url"] = url
	if len(srv.Headers) > 0 {
		entry["headers"] = srv.Headers
	}
	return entry, nil
}
