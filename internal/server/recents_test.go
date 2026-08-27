// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"

	"nui/internal/model"
	"nui/internal/store"
)

func TestTouchRecentSessionDedupesAndCaps(t *testing.T) {
	settings := store.Settings{
		RecentSessionIDs: []string{"c", "b", "a"},
	}
	touchRecentSession(&settings, "b")
	if len(settings.RecentSessionIDs) != 3 {
		t.Fatalf("len = %d, want 3", len(settings.RecentSessionIDs))
	}
	if settings.RecentSessionIDs[0] != "b" {
		t.Fatalf("first = %q, want b", settings.RecentSessionIDs[0])
	}

	for i := 0; i < maxRecentSessions+5; i++ {
		touchRecentSession(&settings, "id-"+string(rune('a'+i%26)))
	}
	if len(settings.RecentSessionIDs) != maxRecentSessions {
		t.Fatalf("len = %d, want cap %d", len(settings.RecentSessionIDs), maxRecentSessions)
	}
}

func TestTouchRecentAgentUpsertsConfig(t *testing.T) {
	settings := store.Settings{
		RecentAgents: []store.RecentAgentEntry{
			{AgentType: "claude-code", WorkingDir: "/old", UsedAt: "2020-01-01T00:00:00Z"},
		},
	}
	touchRecentAgent(&settings, "claude-code", "/new", map[string]any{
		"harnessType": "pi",
	})
	if len(settings.RecentAgents) != 1 {
		t.Fatalf("len = %d, want 1", len(settings.RecentAgents))
	}
	entry := settings.RecentAgents[0]
	if entry.WorkingDir != "/new" {
		t.Fatalf("workingDir = %q, want /new", entry.WorkingDir)
	}
	if entry.AgentConfig["harnessType"] != "pi" {
		t.Fatalf("harnessType = %v, want pi", entry.AgentConfig["harnessType"])
	}
}

func TestMigrateRecentsFromSessions(t *testing.T) {
	settings := store.Settings{}
	sessions := []model.Session{
		{ID: "s1", AgentType: "claude-code", WorkingDir: "/a", CreatedAt: "2026-01-01T00:00:00Z", LastRunAt: "2026-01-03T00:00:00Z"},
		{ID: "s2", AgentType: "codex", WorkingDir: "/b", CreatedAt: "2026-01-02T00:00:00Z"},
		{ID: "s3", AgentType: "claude-code", WorkingDir: "/c", CreatedAt: "2026-01-04T00:00:00Z"},
	}
	migrateRecentsFromSessions(&settings, sessions)
	if len(settings.RecentSessionIDs) != 3 {
		t.Fatalf("recent sessions = %d, want 3", len(settings.RecentSessionIDs))
	}
	if settings.RecentSessionIDs[0] != "s3" {
		t.Fatalf("first session = %q, want s3", settings.RecentSessionIDs[0])
	}
	if len(settings.RecentAgents) != 2 {
		t.Fatalf("recent agents = %d, want 2", len(settings.RecentAgents))
	}
	if settings.RecentAgents[0].AgentType != "claude-code" || settings.RecentAgents[0].WorkingDir != "/c" {
		t.Fatalf("first agent = %+v, want claude-code /c", settings.RecentAgents[0])
	}
}

func TestMigrateRecentsSkipsWhenAlreadyPopulated(t *testing.T) {
	settings := store.Settings{
		RecentSessionIDs: []string{"existing"},
	}
	migrateRecentsFromSessions(&settings, []model.Session{{ID: "s1", AgentType: "codex", CreatedAt: "2026-01-01T00:00:00Z"}})
	if len(settings.RecentSessionIDs) != 1 || settings.RecentSessionIDs[0] != "existing" {
		t.Fatalf("recent sessions changed: %+v", settings.RecentSessionIDs)
	}
}

func TestRemoveRecentSessionID(t *testing.T) {
	settings := store.Settings{
		RecentSessionIDs: []string{"a", "b", "c"},
	}
	removeRecentSessionID(&settings, "b")
	if len(settings.RecentSessionIDs) != 2 || settings.RecentSessionIDs[0] != "a" || settings.RecentSessionIDs[1] != "c" {
		t.Fatalf("ids = %+v, want [a c]", settings.RecentSessionIDs)
	}
}
