// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/hitl"
	"loop/internal/model"
)

func TestWriteClaudeHITLHooksAllowsLoopHitlMCP(t *testing.T) {
	tmp := t.TempDir()
	if err := writeClaudeHITLHooks(tmp); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(tmp, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.Permissions.Allow) != 1 || settings.Permissions.Allow[0] != claudeLoopHitlAllowedTool {
		t.Fatalf("permissions.allow = %v", settings.Permissions.Allow)
	}
}

func TestAppendClaudeInteractiveHitlArgs(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "session")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ".claude.json"), []byte(`{"mcpServers":{"loop-hitl":{}}}`), 0644); err != nil {
		t.Fatal(err)
	}

	req := RunRequest{
		ConfigDir:            configDir,
		HarnessPermissions: hitl.PermissionsInteractive,
	}
	args := appendClaudeInteractiveHitlArgs(nil, req)
	if len(args) != 2 || args[0] != "--allowedTools" || args[1] != claudeLoopHitlAllowedTool {
		t.Fatalf("args = %v", args)
	}

	req.HarnessPermissions = hitl.PermissionsBypass
	if got := appendClaudeInteractiveHitlArgs(nil, req); len(got) != 0 {
		t.Fatalf("bypass should not add allowedTools: %v", got)
	}
}

func TestProvisionClaudeHarnessSkipsHITLSettingsWithoutLoopHitl(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	configDir, err := ProvisionHarnessConfig("no-hitl", "claude-code", HarnessDeps{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no settings.json without loop-hitl, err=%v", err)
	}
}

func TestProvisionClaudeHarnessWritesHITLSettingsWithLoopHitl(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	expanded, err := ExpandHarnessDeps(HarnessDeps{}, nil, "hitl-session", model.ADLDefinition{
		HITL: model.ADLHITL{Mode: hitl.ModeInteractive},
	}, map[string]any{"hitlMode": "interactive"})
	if err != nil {
		t.Fatal(err)
	}
	configDir, err := ProvisionHarnessConfig("hitl-session", "claude-code", expanded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "settings.json")); err != nil {
		t.Fatalf("settings.json: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(configDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		Hooks struct {
			PreToolUse []struct {
				Hooks []struct {
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.Hooks.PreToolUse) == 0 || len(settings.Hooks.PreToolUse[0].Hooks) == 0 {
		t.Fatal("expected PreToolUse hook")
	}
	cmd := settings.Hooks.PreToolUse[0].Hooks[0].Command
	want := filepath.Join(configDir, claudeHitlBridgeScript)
	if cmd != want {
		t.Fatalf("hook command = %q, want %q", cmd, want)
	}
}

func TestAppendHitlSystemPrompt(t *testing.T) {
	base := appendHitlSystemPrompt("You are helpful.")
	if !strings.Contains(base, "ask_user") || !strings.Contains(base, "You are helpful.") {
		t.Fatalf("prompt = %q", base)
	}
	if appendHitlSystemPrompt("") == "" {
		t.Fatal("expected HITL block for empty base prompt")
	}
}

func TestExpandHarnessDepsAppendsHitlSystemPrompt(t *testing.T) {
	expanded, err := ExpandHarnessDeps(HarnessDeps{}, nil, "hitl-session", model.ADLDefinition{
		HITL: model.ADLHITL{Mode: hitl.ModeInteractive},
	}, map[string]any{"hitlMode": "interactive"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(expanded.SystemPrompt, "ask_user") {
		t.Fatalf("system prompt = %q", expanded.SystemPrompt)
	}
}
