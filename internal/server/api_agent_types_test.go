// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"

	"nui/internal/agents"
	"nui/internal/devcontainer"
	"nui/internal/model"
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
		Name:    "Anthropic",
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
