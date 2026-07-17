// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"
	"time"

	"nui/internal/model"
)

func TestEnsureScheduleNextRunAt(t *testing.T) {
	now := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	s := model.Schedule{Interval: "1h", Enabled: true}
	if err := ensureScheduleNextRunAt(&s, now); err != nil {
		t.Fatal(err)
	}
	next, err := time.Parse(time.RFC3339, s.NextRunAt)
	if err != nil {
		t.Fatal(err)
	}
	if !next.Equal(time.Date(2026, 7, 2, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("next = %v, want %v", next, time.Date(2026, 7, 2, 11, 0, 0, 0, time.UTC))
	}
}

func TestSessionHasRunningRun(t *testing.T) {
	resetRunState()
	createRunRecord("sess-run", "run-a", "msg")
	if !sessionHasRunningRun("sess-run") {
		t.Fatal("expected running run")
	}
	finishRunRecord("run-a", RunStatusCompleted, "ok", "")
	if sessionHasRunningRun("sess-run") {
		t.Fatal("expected no running run after finish")
	}
}
