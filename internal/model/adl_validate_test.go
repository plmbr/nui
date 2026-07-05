// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import "testing"

func TestValidateADLDefinitionMinimalAgent(t *testing.T) {
	def := ADLDefinition{
		ID:   "test",
		Name: "Test",
		Harness: ADLHarness{Type: "claude-code"},
	}
	if err := ValidateADLDefinition(def); err != nil {
		t.Fatal(err)
	}
}

func TestValidateADLDefinitionMissingIDAndName(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateADLDefinitionDockerRequiresFields(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "docker-agent",
		Harness: ADLHarness{Type: "docker"},
	})
	if err == nil {
		t.Fatal("expected error for docker without image")
	}
}

func TestValidateADLDefinitionStepDependsOnUnknown(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "wf",
		Harness: ADLHarness{Type: "claude-code"},
		Steps: []ADLStep{
			{Name: "write", DependsOn: []string{"missing"}},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateADLDefinitionHITLStepRequiresBlock(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "wf",
		Harness: ADLHarness{Type: "claude-code"},
		Steps: []ADLStep{
			{Name: "gate", Type: "hitl"},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateADLDefinitionInputReferencesUnknownStep(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "wf",
		Harness: ADLHarness{Type: "claude-code"},
		Steps: []ADLStep{
			{Name: "write", Inputs: []ADLInput{{From: "missing.brief"}}},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateADLDefinitionWorkflowSteps(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "wf",
		Harness: ADLHarness{Type: "claude-code"},
		Steps: []ADLStep{
			{
				Name:    "research",
				Outputs: []ADLOutput{{Name: "brief", Type: "text"}},
			},
			{
				Name:      "write",
				DependsOn: []string{"research"},
				Inputs:    []ADLInput{{From: "research.brief"}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
