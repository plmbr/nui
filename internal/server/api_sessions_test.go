// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"loop/internal/model"
	"loop/internal/store"
)

func TestHandleSessionsListAndCreate(t *testing.T) {
	home := withTempHome(t)
	installTestRemoteAgent(t, home)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	rec := httptest.NewRecorder()
	handleSessions(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}

	createBody := `{"name":"Remote Session","agentType":"test-remote","workingDir":"` + t.TempDir() + `"}`
	req = httptest.NewRequest(http.MethodPost, "/api/sessions", strings.NewReader(createBody))
	rec = httptest.NewRecorder()
	handleSessions(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var created model.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.AgentType != "test-remote" {
		t.Fatalf("session = %+v", created)
	}
}

func TestHandleSessionsCreateRequiresAgentType(t *testing.T) {
	setupTestServerEnv(t)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", strings.NewReader(`{"name":"x"}`))
	rec := httptest.NewRecorder()
	handleSessions(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleSessionRenameDeleteMessages(t *testing.T) {
	withTempHome(t)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}
	seedSession("sess-crud", "Original", testStubAgentType, t.TempDir())

	patchBody := `{"name":"Renamed"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/sessions/sess-crud", strings.NewReader(patchBody))
	rec := httptest.NewRecorder()
	handleSession(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body: %s", rec.Code, rec.Body.String())
	}

	msgs := []model.ChatMessage{
		{ID: "m1", Role: "user", Content: "hello"},
		{ID: "m2", Role: "assistant", Content: "hi"},
	}
	raw, _ := json.Marshal(msgs)
	req = httptest.NewRequest(http.MethodPut, "/api/sessions/sess-crud/messages", bytes.NewReader(raw))
	rec = httptest.NewRecorder()
	handleSessionMessages(rec, req, "sess-crud")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("put messages status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/sessions/sess-crud/messages", nil)
	rec = httptest.NewRecorder()
	handleSessionMessages(rec, req, "sess-crud")
	if rec.Code != http.StatusOK {
		t.Fatalf("get messages status = %d", rec.Code)
	}
	var loaded []model.ChatMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 || loaded[0].Content != "hello" {
		t.Fatalf("messages = %+v", loaded)
	}

	data, err := store.LoadData()
	if err != nil {
		t.Fatal(err)
	}
	if len(data.SessionMessages["sess-crud"]) != 2 {
		t.Fatalf("persisted messages = %+v", data.SessionMessages["sess-crud"])
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/sessions/sess-crud", nil)
	rec = httptest.NewRecorder()
	handleSession(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}

	mu.RLock()
	_, ok := findSession("sess-crud")
	mu.RUnlock()
	if ok {
		t.Fatal("session should be deleted")
	}
}

func TestHandleBulkDeleteSessions(t *testing.T) {
	setupTestServerEnv(t)
	seedSession("sess-a", "A", testStubAgentType, t.TempDir())
	seedSession("sess-b", "B", testStubAgentType, t.TempDir())

	body := `{"ids":["sess-a","missing","sess-b"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/bulk-delete", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleBulkDeleteSessions(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var result map[string][]string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result["deleted"]) != 2 {
		t.Fatalf("deleted = %v", result["deleted"])
	}
}

func TestHandleSessionCreateWithToolApprovalConfig(t *testing.T) {
	home := withTempHome(t)
	installTestRemoteAgent(t, home)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}

	body := `{
		"name":"Policy Session",
		"agentType":"test-remote",
		"workingDir":"` + t.TempDir() + `",
		"agentConfig": {
			"toolApprovalPolicy": "denylist",
			"toolApprovalTools": ["Bash"]
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleSessions(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var created model.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.AgentConfig["toolApprovalPolicy"] != "denylist" {
		t.Fatalf("agentConfig = %+v", created.AgentConfig)
	}
}
