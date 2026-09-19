// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"testing"

	"nui/internal/model"
	"nui/internal/store"
)

func TestApplyBuiltinHarnessModelSettings(t *testing.T) {
	settings := store.Settings{BuiltinHarnessModels: map[string]string{
		"api/anthropic": "claude-custom",
	}}
	builtin := ApplyBuiltinHarnessModelSettings(apiBuiltinAgentDefs[0], settings)
	if builtin.Harness.Model != "claude-custom" {
		t.Fatalf("builtin model = %q", builtin.Harness.Model)
	}

	custom := model.ADLDefinition{
		ID:      "custom",
		Harness: model.ADLHarness{Type: "api", Provider: "anthropic", Model: "authored"},
	}
	custom = ApplyBuiltinHarnessModelSettings(custom, settings)
	if custom.Harness.Model != "authored" {
		t.Fatalf("custom model = %q, want authored", custom.Harness.Model)
	}
}

func TestOrchestratorAndSpecialistShareHarnessModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	settings := store.Settings{
		DefaultHarness: "api/anthropic",
		BuiltinHarnessModels: map[string]string{
			"api/anthropic": "shared-model",
		},
	}
	def := OrchestratorDefinition(settings)
	if def.Harness.Type == "api" && def.Harness.Model != "shared-model" {
		t.Fatalf("nui model = %q, want shared-model", def.Harness.Model)
	}
	specialist := ApplyBuiltinHarnessModelSettings(apiBuiltinAgentDefs[0], settings)
	if specialist.Harness.Model != "shared-model" {
		t.Fatalf("anthropic model = %q, want shared-model", specialist.Harness.Model)
	}
}

func TestEveryBuiltinHarnessSupportsModel(t *testing.T) {
	for _, def := range BuiltinAgentDefs() {
		ref := HarnessRefForDef(def)
		if !BuiltinHarnessSupportsModel(ref) {
			t.Fatalf("%q should support model selection", ref)
		}
	}
}

func TestHarnessRequiresModel(t *testing.T) {
	cases := []struct {
		harness model.ADLHarness
		want    bool
	}{
		{harness: model.ADLHarness{Type: "api", Provider: "anthropic"}, want: true},
		{harness: model.ADLHarness{Type: "api", Provider: "ollama"}, want: true},
		{harness: model.ADLHarness{Type: "antigravity"}, want: true},
		{harness: model.ADLHarness{Type: "claude-code"}, want: false},
	}
	for _, tc := range cases {
		if got := HarnessRequiresModel(tc.harness); got != tc.want {
			t.Fatalf("HarnessRequiresModel(%+v) = %v, want %v", tc.harness, got, tc.want)
		}
	}
}
