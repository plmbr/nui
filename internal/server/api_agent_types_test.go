// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"os"
	"path/filepath"
	"testing"

	"nui/internal/agent"
	"nui/internal/agents"
	"nui/internal/devcontainer"
	"nui/internal/model"
)

func writeFakeCLI(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// withOnlyFakeCLIs sets PATH to dir only and clears NUI_CODEX_PATH so LookPath
// cannot find host agent CLIs or the Codex.app fallback.
func withOnlyFakeCLIs(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("PATH", dir)
	t.Setenv("NUI_CODEX_PATH", filepath.Join(dir, "codex-missing"))
}

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

func TestHarnessAvailable_devcontainerRequiresCLIOnly(t *testing.T) {
	def := model.ADLDefinition{Harness: model.ADLHarness{Type: "devcontainer"}}
	got := harnessAvailable(def)
	want := devcontainer.Available()
	if got != want {
		t.Fatalf("harnessAvailable(devcontainer) = %v, devcontainer.Available() = %v", got, want)
	}
	if got != agentTypeInfoFromDef(def, false).Available {
		t.Fatal("harnessAvailable and agentTypeInfoFromDef disagree for devcontainer")
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

func TestAgentTypeInfoFromDef_apiProvider(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "anthropic",
		Name:    "Claude API",
		Harness: model.ADLHarness{Type: "api", Provider: "anthropic"},
	}
	info := agentTypeInfoFromDef(def, true)
	if info.Harness != "api" {
		t.Fatalf("Harness = %q, want api", info.Harness)
	}
	if info.Provider != "anthropic" {
		t.Fatalf("Provider = %q, want anthropic", info.Provider)
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

func TestAgentTypeInfoFromDef_builtinCLIHarnessPinned(t *testing.T) {
	def := agents.BuiltinAgentDefs()[0] // claude-code
	if def.ID != "claude-code" {
		for _, d := range agents.BuiltinAgentDefs() {
			if d.ID == "claude-code" {
				def = d
				break
			}
		}
	}
	info := agentTypeInfoFromDef(def, true)
	if len(info.AllowedHarnesses) > 1 {
		t.Fatalf("builtin AllowedHarnesses = %v, want at most the matching harness", info.AllowedHarnesses)
	}
	if len(info.AllowedHarnesses) == 1 && info.AllowedHarnesses[0] != "claude-code" {
		t.Fatalf("AllowedHarnesses = %v", info.AllowedHarnesses)
	}
}

func TestAgentTypeInfoFromDef_allowedHarnesses(t *testing.T) {
	def := model.ADLDefinition{
		ID:               "portable",
		Name:             "Portable",
		Harness:          model.ADLHarness{Type: "claude-code"},
		AllowedHarnesses: []string{"pi", "codex"},
	}
	info := agentTypeInfoFromDef(def, false)
	raw := model.NormalizeAllowedHarnesses(def)
	wantSet := map[string]bool{}
	for _, h := range raw {
		probe := def
		probe.Harness.Type = h
		if harnessAvailable(probe) {
			wantSet[h] = true
		}
	}
	if len(info.AllowedHarnesses) != len(wantSet) {
		t.Fatalf("AllowedHarnesses = %v, want available subset of %v", info.AllowedHarnesses, raw)
	}
	for _, h := range info.AllowedHarnesses {
		if !wantSet[h] {
			t.Fatalf("AllowedHarnesses includes unexpected %q", h)
		}
		probe := def
		probe.Harness.Type = h
		if !harnessAvailable(probe) {
			t.Fatalf("AllowedHarnesses includes unavailable %q", h)
		}
	}
}

func TestAgentTypeInfoFromDef_allowedHarnessesFiltersToFakeCLIs(t *testing.T) {
	dir := t.TempDir()
	writeFakeCLI(t, dir, "pi")
	writeFakeCLI(t, dir, "codex")
	withOnlyFakeCLIs(t, dir)
	t.Setenv("NUI_CODEX_PATH", filepath.Join(dir, "codex"))

	def := model.ADLDefinition{
		ID:               "portable",
		Name:             "Portable",
		Harness:          model.ADLHarness{Type: "claude-code"},
		AllowedHarnesses: []string{"pi", "codex"},
	}
	info := agentTypeInfoFromDef(def, false)
	if len(info.AllowedHarnesses) < 2 {
		t.Fatalf("AllowedHarnesses = %v, want pi and codex from fakes", info.AllowedHarnesses)
	}
	found := map[string]bool{}
	for _, h := range info.AllowedHarnesses {
		found[h] = true
	}
	if !found["pi"] || !found["codex"] {
		t.Fatalf("AllowedHarnesses = %v, want to include pi and codex", info.AllowedHarnesses)
	}
	if found["claude-code"] {
		t.Fatalf("claude-code should be unavailable with isolated PATH, got %v", info.AllowedHarnesses)
	}
}

func TestAgentTypeInfoFromDef_allowedHarnessesOmittedExpandsCLI(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "open-cli",
		Name:    "Open CLI",
		Harness: model.ADLHarness{Type: "claude-code"},
	}
	info := agentTypeInfoFromDef(def, false)
	raw := model.NormalizeAllowedHarnesses(def)
	if len(raw) != len(model.CLIHarnessTypes) {
		t.Fatalf("NormalizeAllowedHarnesses = %v, want all CLI types", raw)
	}
	for _, h := range info.AllowedHarnesses {
		probe := def
		probe.Harness.Type = h
		if !harnessAvailable(probe) {
			t.Fatalf("AllowedHarnesses includes unavailable %q", h)
		}
	}
	if agent.CLIAvailable("claude-code") {
		if len(info.AllowedHarnesses) < 1 || info.AllowedHarnesses[0] != "claude-code" {
			t.Fatalf("first = %v, want claude-code when available", info.AllowedHarnesses)
		}
	}
	for _, h := range raw {
		probe := def
		probe.Harness.Type = h
		if !harnessAvailable(probe) {
			continue
		}
		found := false
		for _, got := range info.AllowedHarnesses {
			if got == h {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing available harness %q in %v", h, info.AllowedHarnesses)
		}
	}
}

func TestAgentTypeInfoFromDef_skills(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "skills-agent",
		Name:    "Skills Agent",
		Harness: model.ADLHarness{Type: "claude-code"},
		AIAssets: model.ADLAIAssets{
			Skills: []model.ADLSkill{
				{Name: "code-review", Path: "./skills/code-review"},
				{Name: "commit-helper", Ref: "commit-helper"},
			},
		},
		Steps: []model.ADLStep{{
			AIAssets: model.ADLAIAssets{
				Skills: []model.ADLSkill{
					{Name: "step-skill", Path: "./skills/step-skill"},
					{Name: "code-review", Path: "./skills/code-review"},
				},
			},
		}},
	}
	info := agentTypeInfoFromDef(def, false)
	want := []string{"code-review", "commit-helper", "step-skill", "create-agent", "visualize"}
	if len(info.Skills) != len(want) {
		t.Fatalf("Skills = %v, want %v", info.Skills, want)
	}
	for i, name := range want {
		if info.Skills[i] != name {
			t.Fatalf("Skills[%d] = %q, want %q", i, info.Skills[i], name)
		}
	}
}

func TestSkillNamesFromADL_legacySkill(t *testing.T) {
	def := model.ADLDefinition{
		Skill: "./skills/code-review/SKILL.md",
	}
	got := skillNamesFromADL(def)
	if len(got) != 3 || got[0] != "code-review" || got[1] != "create-agent" || got[2] != "visualize" {
		t.Fatalf("skillNamesFromADL() = %v, want [code-review create-agent visualize]", got)
	}
}

func TestSkillNamesFromADL_includesBuiltinSkills(t *testing.T) {
	got := skillNamesFromADL(model.ADLDefinition{})
	if len(got) == 0 {
		t.Fatal("expected builtin skills")
	}
	found := false
	for _, name := range got {
		if name == "create-agent" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("skillNamesFromADL() = %v, want create-agent", got)
	}
}

func TestBuiltinAgentDefs_havePromptSuggestions(t *testing.T) {
	for _, def := range agents.BuiltinAgentDefs() {
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
