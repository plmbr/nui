// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/store"
)

func testHome(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
}

func TestWriteAndReadUser(t *testing.T) {
	testHome(t)
	if err := WriteUser("prefers concise answers"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadUser()
	if err != nil {
		t.Fatal(err)
	}
	if got != "prefers concise answers" {
		t.Fatalf("ReadUser() = %q", got)
	}
}

func TestWriteAndReadAgent(t *testing.T) {
	testHome(t)
	if err := WriteAgent("my-reviewer", "always check tests"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadAgent("my-reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if got != "always check tests" {
		t.Fatalf("ReadAgent() = %q", got)
	}
}

func TestUpdateAppend(t *testing.T) {
	testHome(t)
	if _, err := Update("user", "", "line one", "replace"); err != nil {
		t.Fatal(err)
	}
	path, err := Update("user", "", "line two", "append")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadUser()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "line one") || !strings.Contains(got, "line two") {
		t.Fatalf("append result = %q", got)
	}
	if !strings.HasSuffix(path, "user.md") {
		t.Fatalf("path = %q", path)
	}
}

func TestUserModeDefaultsManual(t *testing.T) {
	if UserMode(store.Settings{}) != ModeManual {
		t.Fatalf("UserMode() = %q, want manual", UserMode(store.Settings{}))
	}
}

func TestUserModeLegacyDisabled(t *testing.T) {
	disabled := false
	if UserMode(store.Settings{MemoryUserEnabled: &disabled}) != ModeDisabled {
		t.Fatal("expected legacy disabled user mode")
	}
}

func TestUserModeExplicit(t *testing.T) {
	if UserMode(store.Settings{MemoryUserMode: ModeAuto}) != ModeAuto {
		t.Fatal("expected auto")
	}
}

func TestAgentModeLegacyDisabled(t *testing.T) {
	if AgentMode(store.Settings{MemoryAgentsEnabled: map[string]bool{"demo": false}}, "demo") != ModeDisabled {
		t.Fatal("expected legacy disabled agent mode")
	}
}

func TestPromptAppendixRespectsDisabled(t *testing.T) {
	testHome(t)
	if err := WriteUser("user fact"); err != nil {
		t.Fatal(err)
	}
	if err := WriteAgent("demo", "agent fact"); err != nil {
		t.Fatal(err)
	}

	settings := store.Settings{
		MemoryUserMode:   ModeDisabled,
		MemoryAgentsMode: map[string]string{"demo": ModeDisabled},
	}
	if appendix := PromptAppendix(settings, "demo"); appendix != "" {
		t.Fatalf("expected empty appendix when disabled, got %q", appendix)
	}
}

func TestPromptAppendixManualInjects(t *testing.T) {
	testHome(t)
	if err := WriteUser("user fact"); err != nil {
		t.Fatal(err)
	}
	if err := WriteAgent("demo", "agent fact"); err != nil {
		t.Fatal(err)
	}

	settings := store.Settings{MemoryUserMode: ModeManual, MemoryAgentsMode: map[string]string{"demo": ModeManual}}
	appendix := PromptAppendix(settings, "demo")
	if !strings.Contains(appendix, "## User memory") || !strings.Contains(appendix, "user fact") {
		t.Fatalf("missing user memory: %q", appendix)
	}
	if !strings.Contains(appendix, "## Agent memory") || !strings.Contains(appendix, "agent fact") {
		t.Fatalf("missing agent memory: %q", appendix)
	}
}

func TestAutoSaveAppendix(t *testing.T) {
	settings := store.Settings{MemoryAgentsMode: map[string]string{"demo": ModeAuto}}
	appendix := AutoSaveAppendix(settings, "demo")
	if !strings.Contains(appendix, "auto-save") || !strings.Contains(appendix, "update_memory") {
		t.Fatalf("missing auto appendix: %q", appendix)
	}
}

func TestRememberSkillNeeded(t *testing.T) {
	if RememberSkillNeeded(store.Settings{MemoryUserMode: ModeDisabled, MemoryAgentsMode: map[string]string{"a": ModeDisabled}}, "a") {
		t.Fatal("expected false when fully disabled")
	}
	if !RememberSkillNeeded(store.Settings{MemoryUserMode: ModeManual}, "a") {
		t.Fatal("expected true when user manual")
	}
}

func TestCanWriteScope(t *testing.T) {
	settings := store.Settings{MemoryUserMode: ModeDisabled, MemoryAgentsMode: map[string]string{"demo": ModeManual}}
	if CanWriteScope("user", settings, "demo") {
		t.Fatal("user write should be blocked")
	}
	if !CanWriteScope("agent", settings, "demo") {
		t.Fatal("agent write should be allowed")
	}
}
