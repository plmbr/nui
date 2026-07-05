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
	if err := os.WriteFile(filepath.Join(tmp, ".claude.json"), []byte(`{"mcpServers":{"loop-hitl":{},"loop-viz":{}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeClaudeSessionSettings(tmp, HarnessDeps{}); err != nil {
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
	if len(settings.Permissions.Allow) != 3 {
		t.Fatalf("permissions.allow = %v", settings.Permissions.Allow)
	}
	if _, err := os.Stat(filepath.Join(tmp, claudeVizBridgeScript)); err != nil {
		t.Fatalf("expected viz bridge script: %v", err)
	}
}

func TestWriteClaudeSessionSettingsToolApprovalAll(t *testing.T) {
	tmp := t.TempDir()
	if err := writeClaudeSessionSettings(tmp, HarnessDeps{
		ToolApprovalPolicy: hitl.ToolApprovalAll,
	}); err != nil {
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
	if len(settings.Permissions.Allow) != 1 || settings.Permissions.Allow[0] != "*" {
		t.Fatalf("permissions.allow = %v", settings.Permissions.Allow)
	}
}

func TestWriteClaudeSessionSettingsToolApprovalAllowlist(t *testing.T) {
	tmp := t.TempDir()
	if err := writeClaudeSessionSettings(tmp, HarnessDeps{
		ToolApprovalPolicy: hitl.ToolApprovalAllowlist,
		ToolApprovalTools:  []string{"Read", "Grep"},
	}); err != nil {
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
	if len(settings.Permissions.Allow) != 2 {
		t.Fatalf("permissions.allow = %v", settings.Permissions.Allow)
	}
}

func TestExpandHarnessDepsSetsToolApprovals(t *testing.T) {
	expanded, err := ExpandHarnessDeps(HarnessDeps{}, nil, "session", model.ADLDefinition{
		ToolApprovals: model.ADLToolApprovals{
			Policy: hitl.ToolApprovalDenylist,
			Tools:  []string{"Bash"},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if expanded.ToolApprovalPolicy != hitl.ToolApprovalDenylist {
		t.Fatalf("policy = %q", expanded.ToolApprovalPolicy)
	}
	if len(expanded.ToolApprovalTools) != 1 {
		t.Fatalf("tools = %v", expanded.ToolApprovalTools)
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

func TestProvisionClaudeHarnessWritesVizPermissionsWithoutHitl(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	expanded, err := ExpandHarnessDeps(HarnessDeps{}, nil, "viz-only", model.ADLDefinition{
		HITL: model.ADLHITL{Mode: hitl.ModeOff},
	}, map[string]any{"hitlMode": "off"})
	if err != nil {
		t.Fatal(err)
	}
	configDir, err := ProvisionHarnessConfig("viz-only", "claude-code", expanded)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(configDir, "settings.json"))
	if err != nil {
		t.Fatalf("expected settings.json for loop-viz permissions: %v", err)
	}
	var settings struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.Permissions.Allow) != 2 {
		t.Fatalf("permissions.allow = %v", settings.Permissions.Allow)
	}
	if _, err := os.Stat(filepath.Join(configDir, claudeVizBridgeScript)); err != nil {
		t.Fatalf("expected viz bridge script: %v", err)
	}
	var hookSettings struct {
		Hooks struct {
			PreToolUse []struct {
				Matcher string `json:"matcher"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &hookSettings); err != nil {
		t.Fatal(err)
	}
	if len(hookSettings.Hooks.PreToolUse) != 1 || hookSettings.Hooks.PreToolUse[0].Matcher != "Skill|Bash" {
		t.Fatalf("PreToolUse hooks = %+v", hookSettings.Hooks.PreToolUse)
	}
	if _, err := os.Stat(filepath.Join(configDir, claudeHitlBridgeScript)); !os.IsNotExist(err) {
		t.Fatalf("expected no HITL bridge without loop-hitl, err=%v", err)
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
