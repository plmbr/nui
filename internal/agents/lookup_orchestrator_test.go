// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"os"
	"path/filepath"
	"testing"

	"nui/internal/model"
	"nui/internal/store"
)

func TestValidateOrchestratorRefsBuiltin(t *testing.T) {
	err := ValidateOrchestratorRefs(model.ADLDefinition{
		ID:      "council",
		Harness: model.ADLHarness{Type: "claude-code"},
		Orchestration: &model.ADLOrchestration{
			Type:    model.OrchestrationTypeCouncil,
			Members: []model.ADLOrchestrationMember{{Agent: "claude-code"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateOrchestratorRefsSelfReference(t *testing.T) {
	err := ValidateOrchestratorRefs(model.ADLDefinition{
		ID:      "council",
		Harness: model.ADLHarness{Type: "claude-code"},
		Orchestration: &model.ADLOrchestration{
			Type:    model.OrchestrationTypeCouncil,
			Members: []model.ADLOrchestrationMember{{Agent: "council"}},
		},
	})
	if err == nil {
		t.Fatal("expected self-reference error")
	}
}

func TestValidateOrchestratorRefsUnknown(t *testing.T) {
	err := ValidateOrchestratorRefs(model.ADLDefinition{
		ID:      "council",
		Harness: model.ADLHarness{Type: "claude-code"},
		Orchestration: &model.ADLOrchestration{
			Type:    model.OrchestrationTypeCouncil,
			Members: []model.ADLOrchestrationMember{{Agent: "no-such-agent-xyz"}},
		},
	})
	if err == nil {
		t.Fatal("expected unknown agent error")
	}
}

func TestValidateOrchestratorRefsNestedOrchestrationRejected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir, err := store.AgentsDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `adl: "1.0"
id: nested-council
name: Nested
harness:
  type: claude-code
orchestration:
  type: council
  members:
    - agent: claude-code
`
	if err := os.WriteFile(filepath.Join(dir, "nested-council.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err = ValidateOrchestratorRefs(model.ADLDefinition{
		ID:      "outer",
		Harness: model.ADLHarness{Type: "claude-code"},
		Orchestration: &model.ADLOrchestration{
			Type:    model.OrchestrationTypeCouncil,
			Members: []model.ADLOrchestrationMember{{Agent: "nested-council"}},
		},
	})
	if err == nil {
		t.Fatal("expected nested orchestration error")
	}
}
