// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nui/internal/agent"
	"nui/internal/agents"
)

func TestCreateOrchestratorSessionAllowed(t *testing.T) {
	setupTestServerEnv(t)
	s, err := createSession("", t.TempDir(), agents.OrchestratorAgentID, nil)
	if err != nil {
		t.Fatalf("create orchestrator session: %v", err)
	}
	if s.AgentType != agents.OrchestratorAgentID {
		t.Fatalf("agentType = %q", s.AgentType)
	}
}

func TestOrchestratorInAgentTypes(t *testing.T) {
	setupTestServerEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/api/agent-types", nil)
	rec := httptest.NewRecorder()
	handleAgentTypes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var types []AgentTypeInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &types); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, info := range types {
		if info.ID == agents.OrchestratorAgentID {
			found = true
			if !info.IsBuiltin {
				t.Fatal("expected orchestrator to be builtin")
			}
		}
	}
	if !found {
		t.Fatalf("orchestrator %q should be listed", agents.OrchestratorAgentID)
	}
}

func TestLookupOrchestrator(t *testing.T) {
	def, ok := agents.LookupDefinition(agents.OrchestratorAgentID)
	if !ok {
		t.Fatal("expected orchestrator in lookup")
	}
	if def.ID != agents.OrchestratorAgentID {
		t.Fatalf("id = %q", def.ID)
	}
}

func TestOrchestratorSavedAgent(t *testing.T) {
	if !orchestratorSavedAgent(agent.Event{Type: agent.EventToolCallResult, ToolName: "save_agent"}) {
		t.Fatal("expected save_agent tool result")
	}
	if !orchestratorSavedAgent(agent.Event{Type: agent.EventToolCallResult, ToolName: "mcp__nui-agent__save_agent"}) {
		t.Fatal("expected namespaced save_agent tool result")
	}
	if orchestratorSavedAgent(agent.Event{Type: agent.EventToolCallResult, ToolName: "launch_session"}) {
		t.Fatal("launch_session should not count as save_agent")
	}
}

func TestParseLaunchSessionToolResult(t *testing.T) {
	raw := `{
		"session": {"id":"s1","name":"Test","agentType":"anthropic","workingDir":"/tmp","createdAt":"2026-01-01T00:00:00Z"},
		"prompt": "hello"
	}`
	result, ok := parseLaunchSessionToolResult(raw)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if result.Session.ID != "s1" {
		t.Fatalf("session id = %q", result.Session.ID)
	}
	if result.Prompt != "hello" {
		t.Fatalf("prompt = %q", result.Prompt)
	}
}

func TestHarnessResolveAPI(t *testing.T) {
	h, err := agents.HarnessFromRef("api/anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if h.Type != "api" || h.Provider != "anthropic" {
		t.Fatalf("harness = %+v", h)
	}
}

func TestCreateSessionRejectsHiddenInternalAgent(t *testing.T) {
	setupTestServerEnv(t)
	if len(agents.InternalAgentDefs()) == 0 {
		t.Skip("no hidden internal agents configured")
	}
	id := agents.InternalAgentDefs()[0].ID
	_, err := createSession("", t.TempDir(), id, nil)
	if err == nil {
		t.Fatal("expected error creating session with hidden internal agent")
	}
}
