// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"testing"

	"nui/internal/hitl"
)

func TestBuiltinAgentDefsIncludesCLIHarnesses(t *testing.T) {
	defs := BuiltinAgentDefs()
	ids := map[string]bool{}
	for _, def := range defs {
		ids[def.ID] = true
	}
	for _, id := range CLIHarnessTypes {
		if !ids[id] {
			t.Fatalf("missing CLI builtin %q", id)
		}
	}
}

func TestBuiltinAgentDefsCatalogComplete(t *testing.T) {
	want := []string{
		"claude-code", "pi", "codex", "opencode",
		"anthropic", "openai", "gemini", "openrouter", "ollama",
		NuiAgentID,
	}
	defs := BuiltinAgentDefs()
	got := make(map[string]bool, len(defs))
	for _, def := range defs {
		if got[def.ID] {
			t.Fatalf("duplicate builtin id %q", def.ID)
		}
		got[def.ID] = true
	}
	if len(defs) != len(want) {
		t.Fatalf("builtin count = %d, want %d (%v)", len(defs), len(want), got)
	}
	for _, id := range want {
		if !got[id] {
			t.Fatalf("missing builtin %q", id)
		}
	}
}

func TestBuiltinCLIHarnessDefaults(t *testing.T) {
	defs := map[string]struct {
		harnessType string
		perms       string
	}{
		"claude-code": {harnessType: "claude-code", perms: hitl.PermissionsBypass},
		"pi":          {harnessType: "pi"},
		"codex":       {harnessType: "codex", perms: hitl.PermissionsBypass},
		"opencode":    {harnessType: "opencode"},
	}
	for _, def := range BuiltinAgentDefs() {
		want, ok := defs[def.ID]
		if !ok {
			continue
		}
		if def.Harness.Type != want.harnessType {
			t.Fatalf("%s harness type = %q, want %q", def.ID, def.Harness.Type, want.harnessType)
		}
		if def.Harness.Sandbox != "none" {
			t.Fatalf("%s sandbox = %q, want none", def.ID, def.Harness.Sandbox)
		}
		if def.Harness.Permissions != want.perms {
			t.Fatalf("%s permissions = %q, want %q", def.ID, def.Harness.Permissions, want.perms)
		}
		if !def.WorkingDirInput {
			t.Fatalf("%s should allow working dir input", def.ID)
		}
		if len(def.PromptSuggestions) == 0 {
			t.Fatalf("%s missing prompt suggestions", def.ID)
		}
	}
}

func TestCLIHarnessTypesMatchBuiltinDefs(t *testing.T) {
	if len(CLIHarnessTypes) != len(builtinAgentDefs) {
		t.Fatalf("CLIHarnessTypes len = %d, builtinAgentDefs len = %d", len(CLIHarnessTypes), len(builtinAgentDefs))
	}
	for i, id := range CLIHarnessTypes {
		if builtinAgentDefs[i].ID != id {
			t.Fatalf("CLIHarnessTypes[%d] = %q, builtinAgentDefs[%d].ID = %q", i, id, i, builtinAgentDefs[i].ID)
		}
		if builtinAgentDefs[i].Harness.Type != id {
			t.Fatalf("%s harness type = %q", id, builtinAgentDefs[i].Harness.Type)
		}
	}
}

func TestAPIBuiltinAgentsOmitWorkingDirInput(t *testing.T) {
	for _, def := range apiBuiltinAgentDefs {
		if def.WorkingDirInput {
			t.Fatalf("%s should not request working directory input (API agents use isolated workspaces)", def.ID)
		}
		if def.Harness.Type != "api" {
			t.Fatalf("%s harness type = %q, want api", def.ID, def.Harness.Type)
		}
	}
	nui := orchestratorAgentDef()
	if nui.WorkingDirInput {
		t.Fatal("nui should not request working directory input")
	}
}
