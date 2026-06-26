// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"

	"loop/internal/model"
)

func TestHarnessAvailable_builtinCLIHarnesses(t *testing.T) {
	builtins := []model.ADLDefinition{
		{Harness: model.ADLHarness{Type: "claude-code"}},
		{Harness: model.ADLHarness{Type: "pi"}},
		{Harness: model.ADLHarness{Type: "codex"}},
		{Harness: model.ADLHarness{Type: "opencode"}},
	}
	for _, def := range builtins {
		got := harnessAvailable(def)
		want := agentTypeInfoFromDef(def, true).Available
		if got != want {
			t.Fatalf("harnessAvailable(%q) = %v, agentTypeInfoFromDef available = %v", def.Harness.Type, got, want)
		}
	}
}

func TestHarnessAvailable_nonCLIHarnessesAlwaysAvailable(t *testing.T) {
	for _, harnessType := range []string{"docker", "remote", "extension"} {
		def := model.ADLDefinition{Harness: model.ADLHarness{Type: harnessType}}
		if !harnessAvailable(def) {
			t.Fatalf("expected %q harness to be available", harnessType)
		}
	}
}

func TestAgentTypeInfoFromDef_marksBuiltinAvailabilityFromHarness(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "claude-code",
		Name:    "Claude Code",
		Harness: model.ADLHarness{Type: "claude-code"},
	}
	info := agentTypeInfoFromDef(def, true)
	if info.Available != harnessAvailable(def) {
		t.Fatalf("Available = %v, want %v", info.Available, harnessAvailable(def))
	}
	if !info.IsBuiltin {
		t.Fatal("expected builtin flag")
	}
}

func TestAgentTypeInfoFromDef_userAgentCLIHarnessUsesAvailability(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "my-claude",
		Name:    "My Claude",
		Harness: model.ADLHarness{Type: "claude-code"},
	}
	info := agentTypeInfoFromDef(def, false)
	if info.Available != harnessAvailable(def) {
		t.Fatalf("Available = %v, want %v", info.Available, harnessAvailable(def))
	}
}
