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

func TestAgentTypeInfoFromDef_promptSuggestions(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "suggest-agent",
		Name:    "Suggest Agent",
		Harness: model.ADLHarness{Type: "claude-code"},
		PromptSuggestions: []model.ADLPromptSuggestion{
			{Title: "Review", Prompt: "Review the code."},
		},
	}
	info := agentTypeInfoFromDef(def, false)
	if len(info.PromptSuggestions) != 1 {
		t.Fatalf("PromptSuggestions = %+v", info.PromptSuggestions)
	}
	if info.PromptSuggestions[0].Title != "Review" || info.PromptSuggestions[0].Prompt != "Review the code." {
		t.Fatalf("PromptSuggestions[0] = %+v", info.PromptSuggestions[0])
	}
}

func TestBuiltinAgentDefs_havePromptSuggestions(t *testing.T) {
	for _, def := range builtinAgentDefs {
		if len(def.PromptSuggestions) < 2 {
			t.Fatalf("%q: expected at least 2 promptSuggestions, got %d", def.ID, len(def.PromptSuggestions))
		}
		for _, s := range def.PromptSuggestions {
			if s.Title == "" || s.Prompt == "" {
				t.Fatalf("%q: suggestion missing title or prompt: %+v", def.ID, s)
			}
		}
	}
}
