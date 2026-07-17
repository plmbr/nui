// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package nuiclient

import "testing"

func TestPickDefaultAgentTypeID(t *testing.T) {
	agents := []AgentType{
		{ID: "custom", Available: true},
		{ID: "claude-code", Available: true, IsBuiltin: true},
		{ID: "pi", Available: false, IsBuiltin: true},
	}

	id, err := pickDefaultAgentTypeID(agents, "custom")
	if err != nil || id != "custom" {
		t.Fatalf("preferred = %q, err = %v", id, err)
	}

	id, err = pickDefaultAgentTypeID(agents, "missing")
	if err != nil || id != "claude-code" {
		t.Fatalf("builtin fallback = %q, err = %v", id, err)
	}

	apiAgents := []AgentType{
		{ID: "claude-code", Available: true, IsBuiltin: true},
		{ID: "anthropic", Available: true, IsBuiltin: true},
	}
	id, err = pickDefaultAgentTypeID(apiAgents, "")
	if err != nil || id != "anthropic" {
		t.Fatalf("api builtin preference = %q, err = %v", id, err)
	}

	_, err = pickDefaultAgentTypeID([]AgentType{{ID: "x", Available: false}}, "")
	if err == nil {
		t.Fatal("expected error when none available")
	}
}
