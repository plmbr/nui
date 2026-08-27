// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"nui/internal/model"
	"nui/internal/store"
)

const maxRecentSessions = 20
const maxRecentAgents = 20

func touchRecentSession(settings *store.Settings, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	filtered := make([]string, 0, len(settings.RecentSessionIDs)+1)
	for _, id := range settings.RecentSessionIDs {
		if id != sessionID {
			filtered = append(filtered, id)
		}
	}
	settings.RecentSessionIDs = append([]string{sessionID}, filtered...)
	if len(settings.RecentSessionIDs) > maxRecentSessions {
		settings.RecentSessionIDs = settings.RecentSessionIDs[:maxRecentSessions]
	}
}

func touchRecentAgent(settings *store.Settings, agentType, workingDir string, agentConfig map[string]any) {
	agentType = strings.TrimSpace(agentType)
	if agentType == "" {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	entry := store.RecentAgentEntry{
		AgentType:   agentType,
		WorkingDir:  workingDir,
		AgentConfig: cloneAgentConfig(agentConfig),
		UsedAt:      now,
	}
	filtered := make([]store.RecentAgentEntry, 0, len(settings.RecentAgents))
	for _, item := range settings.RecentAgents {
		if item.AgentType != agentType {
			filtered = append(filtered, item)
		}
	}
	settings.RecentAgents = append([]store.RecentAgentEntry{entry}, filtered...)
	if len(settings.RecentAgents) > maxRecentAgents {
		settings.RecentAgents = settings.RecentAgents[:maxRecentAgents]
	}
}

func removeRecentSessionID(settings *store.Settings, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	filtered := settings.RecentSessionIDs[:0]
	for _, id := range settings.RecentSessionIDs {
		if id != sessionID {
			filtered = append(filtered, id)
		}
	}
	settings.RecentSessionIDs = filtered
}

func cloneAgentConfig(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func sessionActivityTime(s model.Session) string {
	if t := strings.TrimSpace(s.LastRunAt); t != "" {
		return t
	}
	return s.CreatedAt
}

func migrateRecentsFromSessions(settings *store.Settings, sessions []model.Session) {
	if len(settings.RecentSessionIDs) > 0 || len(settings.RecentAgents) > 0 {
		return
	}
	if len(sessions) == 0 {
		return
	}

	sorted := append([]model.Session(nil), sessions...)
	sort.Slice(sorted, func(i, j int) bool {
		return sessionActivityTime(sorted[i]) > sessionActivityTime(sorted[j])
	})

	seenAgents := map[string]struct{}{}
	for _, s := range sorted {
		if len(settings.RecentSessionIDs) < maxRecentSessions {
			settings.RecentSessionIDs = append(settings.RecentSessionIDs, s.ID)
		}
		agentType := strings.TrimSpace(s.AgentType)
		if agentType == "" {
			continue
		}
		if _, ok := seenAgents[agentType]; ok {
			continue
		}
		seenAgents[agentType] = struct{}{}
		if len(settings.RecentAgents) >= maxRecentAgents {
			continue
		}
		settings.RecentAgents = append(settings.RecentAgents, store.RecentAgentEntry{
			AgentType:   agentType,
			WorkingDir:  s.WorkingDir,
			AgentConfig: cloneAgentConfig(s.AgentConfig),
			UsedAt:      sessionActivityTime(s),
		})
	}
}

func recordSessionRecents(s model.Session, settings store.Settings) store.Settings {
	touchRecentSession(&settings, s.ID)
	touchRecentAgent(&settings, s.AgentType, s.WorkingDir, s.AgentConfig)
	return settings
}

func ensureRecentsMigrated(settings store.Settings) (store.Settings, bool) {
	if len(settings.RecentSessionIDs) > 0 || len(settings.RecentAgents) > 0 {
		return settings, false
	}
	mu.RLock()
	sessionSnapshot := append([]model.Session(nil), sessions...)
	mu.RUnlock()
	beforeSessions := len(settings.RecentSessionIDs)
	beforeAgents := len(settings.RecentAgents)
	migrateRecentsFromSessions(&settings, sessionSnapshot)
	return settings, len(settings.RecentSessionIDs) != beforeSessions || len(settings.RecentAgents) != beforeAgents
}

func pruneRecentSessionID(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	settings, err := store.LoadSettings()
	if err != nil {
		return
	}
	before := len(settings.RecentSessionIDs)
	removeRecentSessionID(&settings, sessionID)
	if len(settings.RecentSessionIDs) == before {
		return
	}
	if err := store.SaveSettings(settings); err != nil {
		fmt.Fprintf(os.Stderr, "warn: prune recent session id: %v\n", err)
	}
}
