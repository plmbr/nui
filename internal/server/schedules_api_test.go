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
	"time"

	"loop/internal/model"
	"loop/internal/store"
)

func resetScheduleState(t *testing.T) {
	t.Helper()
	schedulesMu.Lock()
	schedules = nil
	schedulesMu.Unlock()
	resetRunState()
	mu.Lock()
	sessions = nil
	mu.Unlock()
}

func TestHandleSchedulesCreateRejectsNonAutoAgent(t *testing.T) {
	resetScheduleState(t)

	body := `{"name":"Daily","agentType":"claude-code","interval":"1h"}`
	req := httptest.NewRequest(http.MethodPost, "/api/schedules", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleCreateSchedule(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSchedulesCreateAutoAgent(t *testing.T) {
	resetScheduleState(t)
	t.Setenv("HOME", t.TempDir())

	dir, err := store.AgentsDir()
	if err != nil {
		t.Fatal(err)
	}
	agentYAML := `adl: "1.0"
id: auto-test
name: Auto Test
promptMode: auto
harness:
  type: claude-code
`
	if err := os.WriteFile(filepath.Join(dir, "auto-test.yaml"), []byte(agentYAML), 0600); err != nil {
		t.Fatal(err)
	}

	body := `{"name":"Hourly","agentType":"auto-test","interval":"1h"}`
	req := httptest.NewRequest(http.MethodPost, "/api/schedules", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleCreateSchedule(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}

	var created model.Schedule
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.NextRunAt == "" {
		t.Fatalf("unexpected schedule: %+v", created)
	}

	oldNext := created.NextRunAt
	patchBody := `{"name":"Hourly updated","interval":"5m"}`
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/schedules/"+created.ID, strings.NewReader(patchBody))
	patchRec := httptest.NewRecorder()
	handlePatchSchedule(patchRec, patchReq, created.ID)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want 200; body: %s", patchRec.Code, patchRec.Body.String())
	}
	var updated model.Schedule
	if err := json.Unmarshal(patchRec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Hourly updated" || updated.Interval != "5m" || updated.Cron != "" {
		t.Fatalf("unexpected patched schedule: %+v", updated)
	}
	if updated.NextRunAt == "" || updated.NextRunAt == oldNext {
		t.Fatalf("expected nextRunAt to be recomputed, got %q (was %q)", updated.NextRunAt, oldNext)
	}
}

func TestScheduleComputeNextRunAtRecovery(t *testing.T) {
	s := model.Schedule{Interval: "5m", Enabled: true}
	now := time.Now().UTC()
	next, err := s.ComputeNextRunAt(now)
	if err != nil {
		t.Fatal(err)
	}
	if !next.After(now) {
		t.Fatalf("next run should be after now")
	}
}

func TestUpdateSessionLastRunAt(t *testing.T) {
	resetScheduleState(t)
	t.Setenv("HOME", t.TempDir())

	mu.Lock()
	sessions = []model.Session{modelSession("sess-last", "Test", "claude-code", "/tmp")}
	mu.Unlock()

	createRunRecord("sess-last", "run-1", "hello")
	finishRunRecord("run-1", RunStatusCompleted, "done", "")

	mu.RLock()
	s, ok := findSession("sess-last")
	mu.RUnlock()
	if !ok {
		t.Fatal("session missing")
	}
	if s.LastRunAt == "" {
		t.Fatalf("expected lastRunAt to be set, got %+v", s)
	}
}
