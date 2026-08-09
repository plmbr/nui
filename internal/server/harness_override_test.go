// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nui/internal/agents"
	"nui/internal/model"
)

func installPortableCoderAgent(t *testing.T, home string) {
	t.Helper()
	agentsDir := filepath.Join(home, ".nui", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `adl: "1.0"
id: portable-coder
name: Portable Coder
harness:
  type: claude-code
allowedHarnesses:
  - claude-code
  - pi
`
	if err := os.WriteFile(filepath.Join(agentsDir, "portable-coder.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func installPinnedCoderAgent(t *testing.T, home string) {
	t.Helper()
	agentsDir := filepath.Join(home, ".nui", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `adl: "1.0"
id: pinned-coder
name: Pinned Coder
harness:
  type: claude-code
allowedHarnesses:
  - claude-code
`
	if err := os.WriteFile(filepath.Join(agentsDir, "pinned-coder.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCreateSessionRejectsHarnessOverrideWhenPinned(t *testing.T) {
	home := withTempHome(t)
	installPinnedCoderAgent(t, home)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	_, err := createSession("", t.TempDir(), "pinned-coder", map[string]any{
		agents.AgentConfigKeyHarnessType: "pi",
	})
	if err == nil || !strings.Contains(err.Error(), "not in allowedHarnesses") {
		t.Fatalf("expected allowlist reject, got %v", err)
	}
}

func TestCreateSessionRejectsHarnessOutsideAllowlist(t *testing.T) {
	home := withTempHome(t)
	installPortableCoderAgent(t, home)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	_, err := createSession("", t.TempDir(), "portable-coder", map[string]any{
		agents.AgentConfigKeyHarnessType: "docker",
	})
	if err == nil || !strings.Contains(err.Error(), "not a CLI harness") {
		t.Fatalf("expected CLI harness error, got %v", err)
	}
}

func TestResolveSessionADLDefAppliesHarnessOverride(t *testing.T) {
	home := withTempHome(t)
	installPortableCoderAgent(t, home)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	session := model.Session{
		AgentType: "portable-coder",
		AgentConfig: map[string]any{
			agents.AgentConfigKeyHarnessType: "pi",
		},
	}
	def, ok := resolveSessionADLDef(session)
	if !ok {
		t.Fatal("expected definition")
	}
	if def.Harness.Type != "pi" {
		t.Fatalf("Harness.Type = %q, want pi", def.Harness.Type)
	}
}

func TestCreateSessionAllowsHarnessOverrideWhenAvailable(t *testing.T) {
	if !agents.HarnessAvailable("pi") {
		t.Skip("pi harness not available on this system")
	}
	home := withTempHome(t)
	installPortableCoderAgent(t, home)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	s, err := createSession("", t.TempDir(), "portable-coder", map[string]any{
		agents.AgentConfigKeyHarnessType: "pi",
	})
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	def, ok := resolveSessionADLDef(s)
	if !ok || def.Harness.Type != "pi" {
		t.Fatalf("effective harness = %+v ok=%v", def.Harness, ok)
	}
}
