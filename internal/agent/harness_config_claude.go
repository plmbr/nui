// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"nui/internal/model"
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
	if err := writeClaudeSessionSettings(configDir, deps); err != nil {
		return err
	}
	if !deps.UserScope {
		if err := linkClaudeAuthFromUser(configDir); err != nil {
			return err
		}
	}
	if err := installHarnessSkills("claude-code", configDir, deps.WorkingDir, deps.Skills); err != nil {
		return fmt.Errorf("install skills: %w", err)
	}
	if _, err := installHarnessRules("claude-code", configDir, deps.ResolvedRules); err != nil {
		return fmt.Errorf("install rules: %w", err)
	}
	return writeHarnessManifest(configDir, "claude-code", deps, map[string]any{
		"systemPromptFile": claudeSystemPromptFile,
		"rulesDir":         "rules",
		"configEnv":        envClaudeConfigDir,
	})
}

// linkClaudeAuthFromUser symlinks the user's Claude login credentials into the
// session config dir so isolated CLAUDE_CONFIG_DIR sessions stay authenticated.
func linkClaudeAuthFromUser(configDir string) error {
	srcDir, err := userClaudeConfigDir()
	if err != nil {
		return err
	}
	absConfig, err := filepath.Abs(configDir)
	if err != nil {
		return err
	}
	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}
	if absConfig == absSrc {
		return nil
	}
	return linkFileIfMissing(
		filepath.Join(srcDir, ".credentials.json"),
		filepath.Join(configDir, ".credentials.json"),
	)
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

	// Claude Code treats CLAUDE_CONFIG_DIR/.claude.json as its primary config and
	// writes session state into it. Merge mcpServers instead of replacing/deleting
	// the whole file so re-provisioning a turn does not wipe Claude's config.
	cfg, err := readClaudeJSONConfig(cfgPath)
	if err != nil {
		return err
	}
	if len(mcpServers) == 0 {
		if cfg == nil {
			return nil
		}
		delete(cfg, "mcpServers")
		if len(cfg) == 0 {
			// Keep an empty object so Claude still finds a config file.
			cfg = map[string]any{}
		}
	} else {
		if cfg == nil {
			cfg = map[string]any{}
		}
		cfg["mcpServers"] = mcpServers
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}

func readClaudeJSONConfig(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Corrupt/non-JSON config: start fresh rather than blocking provision.
		return map[string]any{}, nil
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
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
		if len(srv.Env) > 0 {
			entry["env"] = srv.Env
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
	if len(srv.Headers) > 0 {
		entry["headers"] = srv.Headers
	}
	return entry, nil
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
