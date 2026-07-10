// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"os"
	"path/filepath"
	"testing"

	"loop/internal/model"
	"loop/internal/store"
)

func TestValidateOrchestratorRefsBuiltin(t *testing.T) {
	err := ValidateOrchestratorRefs(model.ADLDefinition{
		ID:        "triage",
		Harness:   model.ADLHarness{Type: "claude-code"},
		SubAgents: []string{"claude-code"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateOrchestratorRefsSelfReference(t *testing.T) {
	err := ValidateOrchestratorRefs(model.ADLDefinition{
		ID:        "triage",
		Harness:   model.ADLHarness{Type: "claude-code"},
		SubAgents: []string{"triage"},
	})
	if err == nil {
		t.Fatal("expected self-reference error")
	}
}

func TestValidateOrchestratorRefsUnknown(t *testing.T) {
	err := ValidateOrchestratorRefs(model.ADLDefinition{
		ID:        "triage",
		Harness:   model.ADLHarness{Type: "claude-code"},
		SubAgents: []string{"no-such-agent-xyz"},
	})
	if err == nil {
		t.Fatal("expected unknown agent error")
	}
}

func TestValidateOrchestratorRefsCycle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir, err := store.AgentsDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeAgent := func(name, id, sub string) {
		content := `adl: "1.0"
id: ` + id + `
name: ` + name + `
harness:
  type: claude-code
`
		if sub != "" {
			content += `subAgents:
  - ` + sub + `
`
		}
		path := filepath.Join(dir, id+".yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeAgent("Agent A", "agent-a", "agent-b")
	writeAgent("Agent B", "agent-b", "agent-a")

	err = ValidateOrchestratorRefs(model.ADLDefinition{
		ID:        "orchestrator",
		Harness:   model.ADLHarness{Type: "claude-code"},
		SubAgents: []string{"agent-a"},
	})
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
