// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUserPromptAgentType_acceptsUserPrompt(t *testing.T) {
	_, err := resolveUserPromptAgentType("claude-code")
	if err != nil {
		t.Fatalf("expected claude-code to be accepted: %v", err)
	}
}

func TestResolveUserPromptAgentType_rejectsUnknown(t *testing.T) {
	_, err := resolveUserPromptAgentType("not-a-real-agent")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestResolveUserPromptAgentType_rejectsAutoPrompt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	agentsDir := filepath.Join(home, ".loop", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `adl: "1.0"
id: auto-agent
name: Auto Agent
promptMode: auto
harness:
  type: claude-code
`
	if err := os.WriteFile(filepath.Join(agentsDir, "auto-agent.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := resolveUserPromptAgentType("auto-agent")
	if err == nil {
		t.Fatal("expected error for auto prompt agent")
	}
}
