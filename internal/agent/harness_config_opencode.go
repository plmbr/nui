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
		"configEnv":        envOpenCodeConfig,
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
// generate. OPENCODE_CONFIG points at the session opencode.json; OPENCODE_CONFIG_DIR is also
// set to the session directory so plugins and custom agents remain discoverable.
var opencodeUserConfigEntries = []string{
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
	if err := linkUserConfigEntries(srcDir, configDir, opencodeUserConfigEntries); err != nil {
		return err
	}
	// OpenCode stores credentials in the XDG data dir, not the config dir.
	dataDir, err := userOpenCodeDataDir()
	if err != nil {
		return err
	}
	return linkFileIfMissing(filepath.Join(dataDir, "auth.json"), filepath.Join(configDir, "auth.json"))
}

// userOpenCodeConfig loads the user's global opencode config so session config generated
// for OPENCODE_CONFIG keeps their provider and model settings.
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
		cfg, err := parseOpenCodeJSONConfig(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[opencode] ignoring unparsable user config %s: %v\n", name, err)
			continue
		}
		return cfg
	}
	return nil
}

func parseOpenCodeJSONConfig(data []byte) (map[string]any, error) {
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err == nil {
		return cfg, nil
	}
	// opencode.jsonc commonly includes // and /* */ comments.
	stripped := stripJSONC(data)
	if err := json.Unmarshal(stripped, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// stripJSONC removes // line comments and /* block comments */ outside of strings.
func stripJSONC(in []byte) []byte {
	var out []byte
	inString := false
	escaped := false
	for i := 0; i < len(in); i++ {
		c := in[i]
		if inString {
			out = append(out, c)
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == '/' && i+1 < len(in) {
			switch in[i+1] {
			case '/':
				i += 2
				for i < len(in) && in[i] != '\n' {
					i++
				}
				if i < len(in) {
					out = append(out, '\n')
				}
				continue
			case '*':
				i += 2
				for i+1 < len(in) && !(in[i] == '*' && in[i+1] == '/') {
					i++
				}
				i++ // consume '/'
				continue
			}
		}
		out = append(out, c)
	}
	return out
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
