// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"testing"

	"nui/internal/uiaction"
)

func TestParseUIActionsFromToolContent(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"ok": true,
		"actions": []uiaction.Action{
			{Type: uiaction.TypeNavigate, Target: uiaction.TargetCustomize},
			{Type: uiaction.TypeSetTheme, Theme: uiaction.ThemeDark},
		},
	})
	actions, ok := parseUIActionsFromToolContent(string(raw))
	if !ok || len(actions) != 2 {
		t.Fatalf("got ok=%v actions=%+v", ok, actions)
	}
}

func TestParseLaunchSessionToolResultEmptyPrompt(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"session": map[string]any{"id": "sess-1", "agentType": "claude-code"},
		"prompt":  "",
	})
	parsed, ok := parseLaunchSessionToolResult(string(raw))
	if !ok {
		t.Fatal("expected parse ok")
	}
	if parsed.Session.ID != "sess-1" {
		t.Fatalf("session id = %q", parsed.Session.ID)
	}
	if parsed.Prompt != "" {
		t.Fatalf("prompt = %q, want empty", parsed.Prompt)
	}
	if !parsed.launchSeen {
		t.Fatal("expected launchSeen")
	}
}

func TestIsControlUIToolName(t *testing.T) {
	if !isControlUIToolName("nui-orchestrator__control_ui") {
		t.Fatal("expected control_ui match")
	}
	if !isSetExtensionEnabledToolName("mcp__nui-orchestrator__set_extension_enabled") {
		t.Fatal("expected set_extension_enabled match")
	}
}
