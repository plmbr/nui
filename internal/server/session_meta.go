// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"os"

	"loop/internal/model"
	"loop/internal/store"
)

func latestAssistantMessageAt(msgs []model.ChatMessage) string {
	var latest string
	for _, m := range msgs {
		if m.Role != "assistant" {
			continue
		}
		if m.CreatedAt > latest {
			latest = m.CreatedAt
		}
	}
	return latest
}

func latestFinishedRunAt(sessionID string) string {
	var latest string
	for _, rec := range listSessionRuns(sessionID) {
		if rec.FinishedAt != "" && rec.FinishedAt > latest {
			latest = rec.FinishedAt
		}
	}
	return latest
}

func latestRunningRunStartedAt(sessionID string) string {
	for _, rec := range listSessionRuns(sessionID) {
		if rec.Status == RunStatusRunning && rec.StartedAt != "" {
			return rec.StartedAt
		}
	}
	return ""
}

// effectiveLastRunAt returns the best-known last run timestamp for display.
func effectiveLastRunAt(session model.Session) string {
	latest := model.NormalizeTimestamp(session.LastRunAt)
	if t := model.NormalizeTimestamp(latestFinishedRunAt(session.ID)); t != "" && t > latest {
		latest = t
	}
	if t := model.NormalizeTimestamp(latestRunningRunStartedAt(session.ID)); t != "" && t > latest {
		latest = t
	}
	if latest != "" {
		return latest
	}
	mu.RLock()
	msgs := sessionMessages[session.ID]
	mu.RUnlock()
	return model.NormalizeTimestamp(latestAssistantMessageAt(msgs))
}

func enrichSession(s model.Session) model.Session {
	if t := effectiveLastRunAt(s); t != "" {
		s.LastRunAt = t
	}
	return s
}

func enrichSessions(list []model.Session) []model.Session {
	out := make([]model.Session, len(list))
	for i, s := range list {
		out[i] = enrichSession(s)
	}
	return out
}

func backfillSessionsLastRunAt() {
	mu.Lock()
	changed := false
	for i, s := range sessions {
		if s.LastRunAt != "" {
			continue
		}
		t := latestAssistantMessageAt(sessionMessages[s.ID])
		if t == "" {
			continue
		}
		sessions[i].LastRunAt = model.NormalizeTimestamp(t)
		changed = true
	}
	var snapshot store.Data
	if changed {
		snapshot = snapshotData()
	}
	mu.Unlock()
	if changed {
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: backfill session lastRunAt: %v\n", err)
		}
	}
}
