// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/store"
)

func TestHandleSkillsList(t *testing.T) {
	withTempHome(t)
	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	rec := httptest.NewRecorder()
	handleSkills(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "[") {
		t.Fatalf("expected JSON array, got %s", rec.Body.String())
	}
}

func TestHandleMCPServersGetPut(t *testing.T) {
	withTempHome(t)
	req := httptest.NewRequest(http.MethodGet, "/api/mcp-servers", nil)
	rec := httptest.NewRecorder()
	handleMCPServers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}

	putBody := `{"mcpServers":[{"name":"test-server","command":"echo","args":["hi"]}]}`
	req = httptest.NewRequest(http.MethodPut, "/api/mcp-servers", strings.NewReader(putBody))
	rec = httptest.NewRecorder()
	handleMCPServers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d, body: %s", rec.Code, rec.Body.String())
	}

	servers, err := store.LoadMCPServers()
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].Name != "test-server" {
		t.Fatalf("servers = %+v", servers)
	}
}

func TestHandleAgentsCRUD(t *testing.T) {
	home := withTempHome(t)
	agentsDir, err := store.AgentsDir()
	if err != nil {
		t.Fatal(err)
	}
	_ = home

	content := `adl: "1.0"
id: custom-agent
name: Custom Agent
harness:
  type: remote
  host: 127.0.0.1
  port: 9091
`
	createBody := `{"file":"custom-agent.yaml","content":` + mustJSONString(content) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/agents", strings.NewReader(createBody))
	rec := httptest.NewRecorder()
	handleAgents(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	rec = httptest.NewRecorder()
	handleAgents(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "custom-agent") {
		t.Fatalf("list body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/agents/custom-agent.yaml", nil)
	rec = httptest.NewRecorder()
	handleAgentFile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}

	updated := strings.ReplaceAll(content, "Custom Agent", "Updated Agent")
	putBody := `{"content":` + mustJSONString(updated) + `}`
	req = httptest.NewRequest(http.MethodPut, "/api/agents/custom-agent.yaml", strings.NewReader(putBody))
	rec = httptest.NewRecorder()
	handleAgentFile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d, body: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/agents/custom-agent.yaml", nil)
	rec = httptest.NewRecorder()
	handleAgentFile(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(agentsDir, "custom-agent.yaml")); !os.IsNotExist(err) {
		t.Fatal("expected agent file removed")
	}
}

func mustJSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}
