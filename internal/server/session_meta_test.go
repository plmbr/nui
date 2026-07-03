// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"
	"time"

	"loop/internal/model"
)

func TestEffectiveLastRunAtFromMessages(t *testing.T) {
	resetRunState()
	mu.Lock()
	sessionMessages = map[string][]model.ChatMessage{
		"s1": {
			{Role: "user", Content: "hi", CreatedAt: "2026-07-01T10:00:00Z"},
			{Role: "assistant", Content: "hello", CreatedAt: "2026-07-01T10:01:00Z"},
		},
	}
	mu.Unlock()

	s := model.Session{ID: "s1"}
	got := effectiveLastRunAt(s)
	if got != "2026-07-01T10:01:00Z" {
		t.Fatalf("effectiveLastRunAt = %q, want assistant message time", got)
	}
}

func TestEffectiveLastRunAtPrefersPersistedRun(t *testing.T) {
	resetRunState()
	mu.Lock()
	sessions = []model.Session{{ID: "s2", LastRunAt: "2026-07-02T12:00:00Z"}}
	sessionMessages = map[string][]model.ChatMessage{
		"s2": {{Role: "assistant", Content: "old", CreatedAt: "2026-07-01T10:01:00Z"}},
	}
	mu.Unlock()

	got := effectiveLastRunAt(sessions[0])
	if got != "2026-07-02T12:00:00Z" {
		t.Fatalf("effectiveLastRunAt = %q, want persisted lastRunAt", got)
	}
}

func TestEffectiveLastRunAtUsesRunningStartedAt(t *testing.T) {
	resetRunState()
	started := time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339)
	createRunRecord("s3", "run-live", "msg")

	runStoreMu.Lock()
	runRecords["run-live"].StartedAt = started
	runStoreMu.Unlock()

	s := model.Session{ID: "s3"}
	got := effectiveLastRunAt(s)
	if got != started {
		t.Fatalf("effectiveLastRunAt = %q, want running startedAt %q", got, started)
	}
}

func TestBackfillSessionsLastRunAt(t *testing.T) {
	resetRunState()
	t.Setenv("HOME", t.TempDir())

	mu.Lock()
	sessions = []model.Session{modelSession("s-back", "Test", "claude-code", "/tmp")}
	sessionMessages = map[string][]model.ChatMessage{
		"s-back": {{Role: "assistant", Content: "done", CreatedAt: "2026-07-01T15:00:00Z"}},
	}
	mu.Unlock()

	backfillSessionsLastRunAt()

	mu.RLock()
	s, _ := findSession("s-back")
	mu.RUnlock()
	if s.LastRunAt != "2026-07-01T15:00:00Z" {
		t.Fatalf("LastRunAt = %q, want backfilled from messages", s.LastRunAt)
	}
}
