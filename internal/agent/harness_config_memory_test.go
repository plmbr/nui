// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"

	"loop/internal/memory"
	"loop/internal/model"
	"loop/internal/skills"
	"loop/internal/store"
)

func TestExpandHarnessDeps_injectsMemory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := memory.WriteUser("prefers concise answers"); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteAgent("demo-agent", "always run tests"); err != nil {
		t.Fatal(err)
	}

	expanded, err := ExpandHarnessDeps(HarnessDeps{
		SystemPrompt: "Base prompt.",
	}, nil, "mem-session", model.ADLDefinition{
		ID:      "demo-agent",
		Harness: model.ADLHarness{Type: "claude-code"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(expanded.SystemPrompt, "## User memory") {
		t.Fatalf("missing user memory section: %q", expanded.SystemPrompt)
	}
	if !strings.Contains(expanded.SystemPrompt, "prefers concise answers") {
		t.Fatalf("missing user memory body: %q", expanded.SystemPrompt)
	}
	if !strings.Contains(expanded.SystemPrompt, "## Agent memory") {
		t.Fatalf("missing agent memory section: %q", expanded.SystemPrompt)
	}
	if !strings.Contains(expanded.SystemPrompt, "always run tests") {
		t.Fatalf("missing agent memory body: %q", expanded.SystemPrompt)
	}
	hasRemember := false
	for _, skill := range expanded.Skills {
		if skill.Name == skills.RememberSkillName {
			hasRemember = true
			break
		}
	}
	if !hasRemember {
		t.Fatal("expected remember skill attached")
	}
}

func TestExpandHarnessDeps_skipsMemoryWhenDisabled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := memory.WriteUser("should not appear"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSettings(store.Settings{
		MemoryUserMode:   memory.ModeDisabled,
		MemoryAgentsMode: map[string]string{"claude-code": memory.ModeDisabled},
	}); err != nil {
		t.Fatal(err)
	}

	expanded, err := ExpandHarnessDeps(HarnessDeps{
		SystemPrompt: "Base only.",
	}, nil, "mem-off", model.ADLDefinition{
		ID:      "claude-code",
		Harness: model.ADLHarness{Type: "claude-code"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(expanded.SystemPrompt, "should not appear") {
		t.Fatalf("memory injected when disabled: %q", expanded.SystemPrompt)
	}
	for _, skill := range expanded.Skills {
		if skill.Name == skills.RememberSkillName {
			t.Fatal("remember skill should not be attached when memory disabled")
		}
	}
}

func TestExpandHarnessDeps_autoAppendix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := store.SaveSettings(store.Settings{
		MemoryUserMode:   memory.ModeManual,
		MemoryAgentsMode: map[string]string{"demo-agent": memory.ModeAuto},
	}); err != nil {
		t.Fatal(err)
	}

	expanded, err := ExpandHarnessDeps(HarnessDeps{
		SystemPrompt: "Base.",
	}, nil, "mem-auto", model.ADLDefinition{
		ID:      "demo-agent",
		Harness: model.ADLHarness{Type: "claude-code"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(expanded.SystemPrompt, "Memory (auto-save)") {
		t.Fatalf("missing auto-save appendix: %q", expanded.SystemPrompt)
	}
}
