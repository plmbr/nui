// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"testing"

	"loop/internal/agent"
	"loop/internal/model"
)

func TestBuiltinAgentDefsIncludesAPIProviders(t *testing.T) {
	defs := BuiltinAgentDefs()
	ids := map[string]model.ADLDefinition{}
	for _, def := range defs {
		ids[def.ID] = def
	}
	for _, id := range APIBuiltinOrder {
		def, ok := ids[id]
		if !ok {
			t.Fatalf("missing builtin %q", id)
		}
		if def.Harness.Type != "api" {
			t.Fatalf("%s harness type = %q", id, def.Harness.Type)
		}
	}
}

func TestAPIBuiltinOrderMatchesAvailabilityCheck(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-test")
	for _, def := range BuiltinAgentDefs() {
		if def.Harness.Type != "api" {
			continue
		}
		_ = agent.APIHarnessAvailable(def.Harness)
	}
}
