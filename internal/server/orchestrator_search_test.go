// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchOrchestratorAgentsFrom_ranksDescriptionMatch(t *testing.T) {
	candidates := []AgentTypeInfo{
		{
			ID:          "ext:pack/task-planner",
			Label:       "Task Planner Agent",
			Description: "List, create, manage project tasks and interact with task sessions",
			Tags:        []string{"tasks", "planner"},
			Available:   true,
		},
		{
			ID:          "ext:pack/acme-coder",
			Label:       "Acme",
			Description: "General coding agent",
			Tags:        []string{"acme"},
			Available:   true,
		},
		{
			ID:          "ext:pack/acme-test",
			Label:       "Acme Test Agent",
			Description: "Test agent for the Acme runtime",
			Available:   true,
		},
	}
	out := searchOrchestratorAgentsFrom("list my project tasks", 5, candidates)
	if len(out) == 0 {
		t.Fatal("expected ranked results")
	}
	if out[0]["id"] != "ext:pack/task-planner" {
		t.Fatalf("top hit = %v, want task-planner (full=%+v)", out[0]["id"], out)
	}
	score, ok := out[0]["score"].(int)
	if !ok || score <= 0 {
		t.Fatalf("expected positive score, got %v", out[0]["score"])
	}
}

func TestSearchOrchestratorAgentsFrom_respectsLimit(t *testing.T) {
	candidates := []AgentTypeInfo{
		{ID: "a", Label: "Alpha", Description: "alpha helper", Available: true},
		{ID: "b", Label: "Beta", Description: "beta helper", Available: true},
		{ID: "c", Label: "Gamma", Description: "gamma helper", Available: true},
	}
	out := searchOrchestratorAgentsFrom("helper", 2, candidates)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
}

func TestHandleOrchestratorSearchAgents(t *testing.T) {
	setupTestServerEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/api/orchestrator/search-agents?q=claude&limit=3", nil)
	rec := httptest.NewRecorder()
	handleOrchestratorSearchAgents(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected at least one result for claude query")
	}
	if _, ok := out[0]["score"]; !ok {
		t.Fatalf("missing score in %+v", out[0])
	}
}

func TestHandleOrchestratorSearchAgents_requiresQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/orchestrator/search-agents", nil)
	rec := httptest.NewRecorder()
	handleOrchestratorSearchAgents(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestFormatLauncherSearchCandidates(t *testing.T) {
	candidates := []AgentTypeInfo{
		{
			ID:          "claude-code",
			Label:       "Claude Code",
			Description: "Claude Code running as a local subprocess",
			Tags:        []string{"builtin", "cli"},
			Available:   true,
		},
	}
	hits := searchOrchestratorAgentsFrom("claude code", 5, candidates)
	out := formatLauncherSearchCandidatesFromHits(hits)
	if out == "" {
		t.Fatal("expected candidate section")
	}
	if !strings.Contains(out, "Candidate agents") {
		t.Fatalf("missing header: %s", out)
	}
	if !strings.Contains(strings.ToLower(out), "claude") {
		t.Fatalf("expected claude-related candidate: %s", out)
	}
}

func TestMatchOrchestratorAgent_clearTaskDescriptionWinner(t *testing.T) {
	candidates := []AgentTypeInfo{
		{
			ID:          "ext:pack/task-planner",
			Label:       "Task Planner Agent",
			Description: "List, create, manage project tasks and interact with task sessions",
			Tags:        []string{"tasks", "planner"},
			Available:   true,
		},
		{
			ID:          "ext:pack/acme-coder",
			Label:       "Acme",
			Description: "General coding agent",
			Available:   true,
		},
	}
	agent, score, ok := matchOrchestratorAgent("list my project tasks", candidates)
	if !ok {
		t.Fatal("expected clear match")
	}
	if agent.ID != "ext:pack/task-planner" {
		t.Fatalf("got %q score=%d", agent.ID, score)
	}
}
