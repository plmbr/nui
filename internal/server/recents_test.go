// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"

	"nui/internal/model"
	"nui/internal/store"
)

func TestTouchRecentSessionDedupesAndCaps(t *testing.T) {
	st := store.State{
		RecentSessionIDs: []string{"c", "b", "a"},
	}
	touchRecentSession(&st, "b")
	if len(st.RecentSessionIDs) != 3 {
		t.Fatalf("len = %d, want 3", len(st.RecentSessionIDs))
	}
	if st.RecentSessionIDs[0] != "b" {
		t.Fatalf("first = %q, want b", st.RecentSessionIDs[0])
	}

	for i := 0; i < maxRecentSessions+5; i++ {
		touchRecentSession(&st, "id-"+string(rune('a'+i%26)))
	}
	if len(st.RecentSessionIDs) != maxRecentSessions {
		t.Fatalf("len = %d, want cap %d", len(st.RecentSessionIDs), maxRecentSessions)
	}
}

func TestTouchRecentAgentUpsertsConfig(t *testing.T) {
	st := store.State{
		RecentAgents: []store.RecentAgentEntry{
			{AgentType: "claude-code", WorkingDir: "/old", UsedAt: "2020-01-01T00:00:00Z"},
		},
	}
	touchRecentAgent(&st, "claude-code", "/new", map[string]any{
		"harnessType": "pi",
	})
	if len(st.RecentAgents) != 1 {
		t.Fatalf("len = %d, want 1", len(st.RecentAgents))
	}
	entry := st.RecentAgents[0]
	if entry.WorkingDir != "/new" {
		t.Fatalf("workingDir = %q, want /new", entry.WorkingDir)
	}
	if entry.AgentConfig["harnessType"] != "pi" {
		t.Fatalf("harnessType = %v, want pi", entry.AgentConfig["harnessType"])
	}
}

func TestMigrateRecentsFromSessions(t *testing.T) {
	st := store.State{}
	sessions := []model.Session{
		{ID: "s1", AgentType: "claude-code", WorkingDir: "/a", CreatedAt: "2026-01-01T00:00:00Z", LastRunAt: "2026-01-03T00:00:00Z"},
		{ID: "s2", AgentType: "codex", WorkingDir: "/b", CreatedAt: "2026-01-02T00:00:00Z"},
		{ID: "s3", AgentType: "claude-code", WorkingDir: "/c", CreatedAt: "2026-01-04T00:00:00Z"},
	}
	migrateRecentsFromSessions(&st, sessions)
	if len(st.RecentSessionIDs) != 3 {
		t.Fatalf("recent sessions = %d, want 3", len(st.RecentSessionIDs))
	}
	if st.RecentSessionIDs[0] != "s3" {
		t.Fatalf("first session = %q, want s3", st.RecentSessionIDs[0])
	}
	if len(st.RecentAgents) != 2 {
		t.Fatalf("recent agents = %d, want 2", len(st.RecentAgents))
	}
	if st.RecentAgents[0].AgentType != "claude-code" || st.RecentAgents[0].WorkingDir != "/c" {
		t.Fatalf("first agent = %+v, want claude-code /c", st.RecentAgents[0])
	}
}

func TestMigrateRecentsSkipsWhenAlreadyPopulated(t *testing.T) {
	st := store.State{
		RecentSessionIDs: []string{"existing"},
	}
	migrateRecentsFromSessions(&st, []model.Session{{ID: "s1", AgentType: "codex", CreatedAt: "2026-01-01T00:00:00Z"}})
	if len(st.RecentSessionIDs) != 1 || st.RecentSessionIDs[0] != "existing" {
		t.Fatalf("recent sessions changed: %+v", st.RecentSessionIDs)
	}
}

func TestRemoveRecentSessionID(t *testing.T) {
	st := store.State{
		RecentSessionIDs: []string{"a", "b", "c"},
	}
	removeRecentSessionID(&st, "b")
	if len(st.RecentSessionIDs) != 2 || st.RecentSessionIDs[0] != "a" || st.RecentSessionIDs[1] != "c" {
		t.Fatalf("ids = %+v, want [a c]", st.RecentSessionIDs)
	}
}
