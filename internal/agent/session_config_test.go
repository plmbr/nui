// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestUserScopeHarnessConfig(t *testing.T) {
	if UserScopeHarnessConfig(nil) {
		t.Fatal("nil config should be false")
	}
	if UserScopeHarnessConfig(map[string]any{"other": true}) {
		t.Fatal("missing key should be false")
	}
	if !UserScopeHarnessConfig(map[string]any{AgentConfigKeyUserScopeHarness: true}) {
		t.Fatal("expected true")
	}
}

func TestHarnessSupportsUserScope(t *testing.T) {
	if !HarnessSupportsUserScope("claude-code") {
		t.Fatal("claude-code should support user scope")
	}
	if !HarnessSupportsUserScope("codex") {
		t.Fatal("codex should support user scope")
	}
	if HarnessSupportsUserScope("pi") {
		t.Fatal("pi should not support user scope")
	}
	if HarnessSupportsUserScope("opencode") {
		t.Fatal("opencode should not support user scope")
	}
}

func TestAppendClaudeUserScopeArgs(t *testing.T) {
	args := appendClaudeUserScopeArgs(nil, "")
	if len(args) != 2 || args[0] != "--setting-sources" || args[1] != "user,project,local" {
		t.Fatalf("args = %v", args)
	}

	tmp := t.TempDir()
	mcpPath := filepath.Join(tmp, ".claude.json")
	data, err := json.Marshal(map[string]any{
		"mcpServers": map[string]any{"docs": map[string]any{"url": "http://localhost"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mcpPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	args = appendClaudeUserScopeArgs(nil, tmp)
	if len(args) != 4 || args[2] != "--mcp-config" || args[3] != mcpPath {
		t.Fatalf("args = %v", args)
	}
}

func TestApplyCmdEnvOpenCodeSetsConfigFile(t *testing.T) {
	cmd := exec.Command("true")
	applyCmdEnv(cmd, "opencode", "/tmp/session-config", nil, false, "", "")
	m := envMap(cmd.Env)
	if m[envOpenCodeConfig] != "/tmp/session-config/"+opencodeConfigFile {
		t.Fatalf("OPENCODE_CONFIG = %q", m[envOpenCodeConfig])
	}
	if m[envOpenCodeConfigDir] != "/tmp/session-config" {
		t.Fatalf("OPENCODE_CONFIG_DIR = %q", m[envOpenCodeConfigDir])
	}
}

func TestApplyCmdEnvUserScopeSkipsConfigDir(t *testing.T) {
	cmd := exec.Command("true")
	applyCmdEnv(cmd, "claude-code", "/tmp/session-config", nil, true, "", "")
	m := envMap(cmd.Env)
	if _, ok := m["CLAUDE_CONFIG_DIR"]; ok {
		t.Fatalf("CLAUDE_CONFIG_DIR should be omitted: %v", m)
	}
}
