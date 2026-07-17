// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleAgentEvalRunValidation(t *testing.T) {
	home := withTempHome(t)
	agentsDir := filepath.Join(home, ".nui", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "no-evals.yaml"), []byte(`adl: "1.0"
id: no-evals
name: No Evals
harness:
  type: claude-code
`), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("agent not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/agents/missing/evals/run", strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		handleAgentEvalRun(rec, req, "missing")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("agent has no evals", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/agents/no-evals/evals/run", strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		handleAgentEvalRun(rec, req, "no-evals")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandleAgentFileEvalsRunRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/agents/my-agent/evals/run", strings.NewReader(`{}`))
	req.URL.Path = "/api/agents/my-agent/evals/run"
	rec := httptest.NewRecorder()
	handleAgentFile(rec, req)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest {
		// Agent file may not exist; route should reach eval handler (not 405).
		if rec.Code == http.StatusMethodNotAllowed {
			t.Fatalf("unexpected method not allowed for evals/run route")
		}
	}
}
