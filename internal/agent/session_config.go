// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const AgentConfigKeyUserScopeHarness = "userScopeHarnessConfig"

// UserScopeHarnessConfig reports whether a session should load harness user/project
// settings via native CLI options instead of redirecting config through session env vars.
func UserScopeHarnessConfig(cfg map[string]any) bool {
	if cfg == nil {
		return false
	}
	v, _ := cfg[AgentConfigKeyUserScopeHarness].(bool)
	return v
}

// HarnessSupportsUserScope reports whether a harness type exposes CLI flags for
// loading user-scoped settings alongside the working directory.
func HarnessSupportsUserScope(harnessType string) bool {
	switch normalizeHarnessType(harnessType) {
	case "claude-code", "codex":
		return true
	default:
		return false
	}
}

func effectiveUserScopeHarness(harnessType string, userScope bool) bool {
	return userScope && HarnessSupportsUserScope(harnessType)
}

func claudeMCPConfigPath(configDir string) string {
	if configDir == "" {
		return ""
	}
	path := filepath.Join(configDir, ".claude.json")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		MCPServers map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil || len(cfg.MCPServers) == 0 {
		return ""
	}
	return path
}

func claudeSettingsPath(configDir string) string {
	if configDir == "" {
		return ""
	}
	path := filepath.Join(configDir, "settings.json")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ""
	}
	return path
}

func appendClaudeInteractivePermissionArgs(args []string, configDir string, interactive bool) []string {
	if !interactive {
		return args
	}
	args = append(args, "--permission-prompt-tool", "stdio")
	if path := claudeSettingsPath(configDir); path != "" {
		args = append(args, "--settings", path)
	}
	return args
}

func appendClaudeUserScopeArgs(args []string, configDir string) []string {
	args = append(args, "--setting-sources", "user,project,local")
	if path := claudeMCPConfigPath(configDir); path != "" {
		args = append(args, "--mcp-config", path)
	}
	return args
}

func effectiveHarnessConfigBindDir(harnessType, sessionConfigDir string, userScope bool) string {
	if !userScope {
		return harnessConfigBindDir(harnessType, sessionConfigDir)
	}
	switch normalizeHarnessType(harnessType) {
	case "claude-code":
		dir, err := userClaudeConfigDir()
		if err == nil {
			return dir
		}
	case "codex":
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, ".codex")
		}
	}
	return ""
}
