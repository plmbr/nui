// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"

	"nui/internal/model"
)

func TestFilterPublicSessionsHidesCouncilChildren(t *testing.T) {
	list := []model.Session{
		{ID: "parent", Name: "Council"},
		{
			ID:   "child",
			Name: "Council · Claude",
			AgentConfig: map[string]any{
				agentConfigCouncilParent: "parent",
				agentConfigCouncilMember: "claude-code",
			},
		},
	}
	out := filterPublicSessions(list)
	if len(out) != 1 || out[0].ID != "parent" {
		t.Fatalf("got %+v", out)
	}
}

func TestIsCouncilManagedSession(t *testing.T) {
	if isCouncilManagedSession(model.Session{ID: "a"}) {
		t.Fatal("expected false")
	}
	if !isCouncilManagedSession(model.Session{
		AgentConfig: map[string]any{agentConfigCouncilParent: "p"},
	}) {
		t.Fatal("expected true")
	}
}
