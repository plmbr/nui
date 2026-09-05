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

func TestValidateADLDefinitionDevcontainerRequiresContainerPort(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "devcontainer-agent",
		Harness: ADLHarness{Type: "devcontainer"},
	})
	if err == nil {
		t.Fatal("expected error for devcontainer without innerHarness")
	}
}

func TestValidateADLDefinitionDevcontainerValid(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "devcontainer-agent",
		Harness: ADLHarness{Type: "devcontainer", InnerHarness: "claude-code"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateADLDefinitionStepDependsOnUnknown(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "wf",
		Harness: ADLHarness{Type: "claude-code"},
		Orchestration: &ADLOrchestration{
			Type: OrchestrationTypeWorkflow,
			Steps: []ADLStep{
				{Name: "write", DependsOn: []string{"missing"}},
			},
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
		Orchestration: &ADLOrchestration{
			Type: OrchestrationTypeWorkflow,
			Steps: []ADLStep{
				{Name: "gate", Type: "hitl"},
			},
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
		Orchestration: &ADLOrchestration{
			Type: OrchestrationTypeWorkflow,
			Steps: []ADLStep{
				{Name: "write", Inputs: []ADLInput{{From: "missing.brief"}}},
			},
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
		Orchestration: &ADLOrchestration{
			Type: OrchestrationTypeWorkflow,
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
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateADLDefinitionCouncilValid(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "council",
		Harness: ADLHarness{Type: "claude-code"},
		Orchestration: &ADLOrchestration{
			Type:    OrchestrationTypeCouncil,
			Members: []ADLOrchestrationMember{{Agent: "hello-world"}, {Agent: "code-reviewer"}},
			Rounds:  "independent+rebuttal",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateADLDefinitionSubAgentsValid(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "orch",
		Harness: ADLHarness{Type: "claude-code"},
		Orchestration: &ADLOrchestration{
			Type:     OrchestrationTypeSubAgents,
			Members:  []ADLOrchestrationMember{{Agent: "hello-world"}},
			MaxTurns: 10,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateADLDefinitionLegacyTopLevelRejected(t *testing.T) {
	if err := ValidateADLDefinition(ADLDefinition{
		ID:            "bad",
		Harness:       ADLHarness{Type: "claude-code"},
		LegacyCouncil: &ADLCouncil{Members: []ADLOrchestrationMember{{Agent: "hello-world"}}},
	}); err == nil {
		t.Fatal("expected council rejection")
	}
	if err := ValidateADLDefinition(ADLDefinition{
		ID:          "bad",
		Harness:     ADLHarness{Type: "claude-code"},
		LegacySteps: []ADLStep{{Name: "a"}},
	}); err == nil {
		t.Fatal("expected steps rejection")
	}
	if err := ValidateADLDefinition(ADLDefinition{
		ID:              "bad",
		Harness:         ADLHarness{Type: "claude-code"},
		LegacySubAgents: []string{"hello-world"},
	}); err == nil {
		t.Fatal("expected subAgents rejection")
	}
}

func TestValidateADLDefinitionCouncilDuplicate(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:      "bad",
		Harness: ADLHarness{Type: "claude-code"},
		Orchestration: &ADLOrchestration{
			Type:    OrchestrationTypeCouncil,
			Members: []ADLOrchestrationMember{{Agent: "hello-world"}, {Agent: "hello-world"}},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
