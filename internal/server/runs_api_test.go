// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"loop/internal/model"
)

func TestHandleSessionRunsStartAndGet(t *testing.T) {
	resetRunState()
	mu.Lock()
	sessions = []model.Session{modelSession("sess-runs", "Test", "claude-code", "/tmp")}
	mu.Unlock()

	body := `{"message":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/sess-runs/runs", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleSessionRuns(rec, req, "sess-runs")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("start status = %d, want 202; body: %s", rec.Code, rec.Body.String())
	}

	var started map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	runID, _ := started["runId"].(string)
	if runID == "" {
		t.Fatalf("missing runId: %+v", started)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/sessions/sess-runs/runs/"+runID, nil)
	rec = httptest.NewRecorder()
	handleSessionRunsRouter(rec, req, "sess-runs", "runs/"+runID)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var recOut RunRecord
	if err := json.Unmarshal(rec.Body.Bytes(), &recOut); err != nil {
		t.Fatal(err)
	}
	if recOut.RunID != runID {
		t.Fatalf("runId = %q", recOut.RunID)
	}
}

func TestCancelActiveRunByRunID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	registerActiveRun("sess-1", "run-1", cancel)
	if !cancelActiveRun("sess-1", "run-1") {
		t.Fatal("expected cancel to succeed")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected context cancelled")
	}
}

func resetRunState() {
	runStoreMu.Lock()
	runRecords = map[string]*RunRecord{}
	sessionRuns = map[string][]string{}
	runListeners = map[string]map[chan runLogEntry]struct{}{}
	runStoreMu.Unlock()
	activeRunsMu.Lock()
	activeRunsBySession = map[string]*activeRun{}
	activeRunsByID = map[string]*activeRun{}
	activeRunsMu.Unlock()
}
