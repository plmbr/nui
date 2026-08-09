// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"testing"

	"nui/internal/model"
	"nui/internal/store"
)

func TestOrchestratorDefinitionPinsCLIHarness(t *testing.T) {
	def := OrchestratorDefinition(store.Settings{DefaultHarness: "pi"})
	if def.Harness.Type != "pi" {
		// May fall back if pi unavailable; still must not expose a multi-harness allowlist.
		t.Logf("Harness.Type = %q (pi may be unavailable)", def.Harness.Type)
	}
	allowed := model.NormalizeAllowedHarnesses(def)
	if model.IsCLIHarnessType(def.Harness.Type) {
		if len(allowed) != 1 || allowed[0] != def.Harness.Type {
			t.Fatalf("AllowedHarnesses = %v, want singleton [%s]", allowed, def.Harness.Type)
		}
	} else if len(allowed) != 0 {
		t.Fatalf("non-CLI orchestrator allowlist = %v, want empty", allowed)
	}
}

func TestOrchestratorDefinitionAPIHasNoCLIAllowlist(t *testing.T) {
	def := OrchestratorDefinition(store.Settings{DefaultHarness: "api/anthropic"})
	if def.Harness.Type != "api" {
		t.Logf("Harness.Type = %q (api/anthropic may be unavailable)", def.Harness.Type)
	}
	if got := model.NormalizeAllowedHarnesses(def); len(got) != 0 {
		t.Fatalf("API orchestrator allowlist = %v, want empty", got)
	}
}
