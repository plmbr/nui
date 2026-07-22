// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"testing"

	"nui/internal/store"
)

func TestHarnessFromRef_api(t *testing.T) {
	h, err := HarnessFromRef("api/anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if h.Type != "api" || h.Provider != "anthropic" {
		t.Fatalf("harness = %+v", h)
	}
}

func TestHarnessFromRef_cli(t *testing.T) {
	h, err := HarnessFromRef("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if h.Type != "claude-code" {
		t.Fatalf("harness = %+v", h)
	}
}

func TestHarnessFromRef_unknown(t *testing.T) {
	_, err := HarnessFromRef("api/not-a-provider")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsOrchestratorAgent(t *testing.T) {
	if !IsOrchestratorAgent(NuiAgentID) {
		t.Fatal("nui should be recognized as orchestrator")
	}
	if IsOrchestratorAgent("claude-code") {
		t.Fatal("claude-code should not be orchestrator")
	}
	if !IsOrchestratorAgent("nui-orchestrator") {
		t.Fatal("legacy nui-orchestrator id should resolve to nui")
	}
}

func TestIsOrchestratorRoutingTarget(t *testing.T) {
	if IsOrchestratorRoutingTarget(NuiAgentID) {
		t.Fatal("nui should not be a routing target")
	}
	if !IsOrchestratorRoutingTarget("claude-code") {
		t.Fatal("claude-code should be a routing target")
	}
}

func TestOrchestratorInBuiltinAgentDefs(t *testing.T) {
	var found bool
	for _, def := range BuiltinAgentDefs() {
		if def.ID == NuiAgentID {
			found = true
			if def.Name != "nui" {
				t.Fatalf("name = %q, want nui", def.Name)
			}
			break
		}
	}
	if !found {
		t.Fatal("nui should be in builtin agent defs")
	}
}

func TestPickDefaultHarnessRef_emptySettings(t *testing.T) {
	ref := PickDefaultHarnessRef(store.Settings{})
	if ref == "" {
		t.Skip("no harness available in test environment")
	}
	if !HarnessAvailable(ref) {
		t.Fatalf("picked unavailable ref %q", ref)
	}
}
