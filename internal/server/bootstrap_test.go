// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"os"
	"path/filepath"
	"testing"

	"loop/internal/model"
	"loop/internal/store"
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

func TestApplyStartSettings_theme(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := applyStartSettings(StartOptions{Theme: "dark"}); err != nil {
		t.Fatal(err)
	}
	settings, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Theme != "dark" {
		t.Fatalf("theme = %q, want dark", settings.Theme)
	}
}

func TestApplyStartSettings_invalidTheme(t *testing.T) {
	err := applyStartSettings(StartOptions{Theme: "sepia"})
	if err == nil {
		t.Fatal("expected error for invalid theme")
	}
}

func TestApplyStartSettings_defaultAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := applyStartSettings(StartOptions{DefaultAgentType: "claude-code"}); err != nil {
		t.Fatal(err)
	}
	settings, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.DefaultAgentType != "claude-code" {
		t.Fatalf("defaultAgentType = %q, want claude-code", settings.DefaultAgentType)
	}
}

func TestApplyStartSettings_unknownDefaultAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := applyStartSettings(StartOptions{DefaultAgentType: "not-a-real-agent"})
	if err == nil {
		t.Fatal("expected error for unknown default agent")
	}
}

func TestBootstrapFromCLI_noAgentTypeDoesNotCreateSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapFromCLI(StartOptions{}); err != nil {
		t.Fatalf("bootstrapFromCLI: %v", err)
	}

	mu.RLock()
	count := len(sessions)
	mu.RUnlock()
	if count != 0 {
		t.Fatalf("sessions = %d, want 0", count)
	}
}

func TestGetDefaultSession_returnsLastSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	sessions = []model.Session{
		{ID: "sess-1", Name: "First", AgentType: "claude-code"},
		{ID: "sess-2", Name: "Second", AgentType: "claude-code"},
	}
	mu.Unlock()

	if err := store.SaveSettings(store.Settings{
		Theme:          "light",
		LastSessionID:  "sess-2",
		DefaultAgentType: "claude-code",
	}); err != nil {
		t.Fatal(err)
	}

	s, err := getDefaultSession()
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "sess-2" {
		t.Fatalf("session id = %q, want sess-2", s.ID)
	}
}

func TestGetDefaultSession_noSessions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	_, err := getDefaultSession()
	if err == nil {
		t.Fatal("expected error when no sessions exist")
	}
}
