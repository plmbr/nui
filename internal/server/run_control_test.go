// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"loop/internal/model"
)

func TestHandleSessionStop(t *testing.T) {
	mu.Lock()
	sessions = append(sessions, modelSession("sess-stop", "Test", "claude-code", "/tmp"))
	mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/sess-stop/stop", nil)
	rec := httptest.NewRecorder()
	handleSessionStop(rec, req, "sess-stop")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sessions/missing/stop", nil)
	rec = httptest.NewRecorder()
	handleSessionStop(rec, req, "missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing session status = %d, want 404", rec.Code)
	}
}

func modelSession(id, name, agentType, workingDir string) model.Session {
	return model.Session{
		ID:         id,
		Name:       name,
		AgentType:  agentType,
		WorkingDir: workingDir,
	}
}
