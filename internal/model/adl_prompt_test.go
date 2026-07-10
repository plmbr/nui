// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestResolveADLLaunchPrompt(t *testing.T) {
	def := ADLDefinition{DefaultPrompt: "Custom default."}
	if got := ResolveADLLaunchPrompt(def, "override"); got != "override" {
		t.Fatalf("override = %q", got)
	}
	if got := ResolveADLLaunchPrompt(def, ""); got != "Custom default." {
		t.Fatalf("default = %q", got)
	}
	if got := ResolveADLLaunchPrompt(ADLDefinition{}, ""); got != ADLDefaultAutoPrompt {
		t.Fatalf("builtin = %q", got)
	}
}

func TestADLDefinitionYAML_promptMode(t *testing.T) {
	raw := []byte(`adl: "1.0"
name: auto-agent
harness:
  type: claude-code
promptMode: auto
defaultPrompt: Run the daily check.
`)

	var def ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}
	if !IsADLAutoPrompt(def) {
		t.Fatal("expected auto prompt mode")
	}
	if def.DefaultPrompt != "Run the daily check." {
		t.Fatalf("defaultPrompt = %q", def.DefaultPrompt)
	}
}

func TestIsMultiStepWorkflow(t *testing.T) {
	if IsMultiStepWorkflow(ADLDefinition{Harness: ADLHarness{Type: "claude-code"}}) {
		t.Fatal("single-step agent should not be multi-step workflow")
	}
	if !IsMultiStepWorkflow(ADLDefinition{
		Harness: ADLHarness{Type: "claude-code"},
		Steps:   []ADLStep{{Name: "a"}},
	}) {
		t.Fatal("expected multi-step when steps present")
	}
	if !IsMultiStepWorkflow(ADLDefinition{
		Kind:    "workflow",
		Harness: ADLHarness{Type: "claude-code"},
	}) {
		t.Fatal("expected multi-step for kind workflow")
	}
}

func TestIsOrchestratorAgent(t *testing.T) {
	if IsOrchestratorAgent(ADLDefinition{Harness: ADLHarness{Type: "claude-code"}}) {
		t.Fatal("single-step agent should not be orchestrator")
	}
	if !IsOrchestratorAgent(ADLDefinition{
		Harness:   ADLHarness{Type: "claude-code"},
		SubAgents: []string{"hello-world"},
	}) {
		t.Fatal("expected orchestrator when subAgents present")
	}
	if !SkipsHarnessSessionPersistence(ADLDefinition{
		Harness:   ADLHarness{Type: "claude-code"},
		SubAgents: []string{"hello-world"},
	}) {
		t.Fatal("orchestrator should skip top-level harness session persistence")
	}
}
