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

func touchRecentSession(st *store.State, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	filtered := make([]string, 0, len(st.RecentSessionIDs)+1)
	for _, id := range st.RecentSessionIDs {
		if id != sessionID {
			filtered = append(filtered, id)
		}
	}
	st.RecentSessionIDs = append([]string{sessionID}, filtered...)
	if len(st.RecentSessionIDs) > maxRecentSessions {
		st.RecentSessionIDs = st.RecentSessionIDs[:maxRecentSessions]
	}
}

func touchRecentAgent(st *store.State, agentType, workingDir string, agentConfig map[string]any) {
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
	filtered := make([]store.RecentAgentEntry, 0, len(st.RecentAgents))
	for _, item := range st.RecentAgents {
		if item.AgentType != agentType {
			filtered = append(filtered, item)
		}
	}
	st.RecentAgents = append([]store.RecentAgentEntry{entry}, filtered...)
	if len(st.RecentAgents) > maxRecentAgents {
		st.RecentAgents = st.RecentAgents[:maxRecentAgents]
	}
}

func removeRecentSessionID(st *store.State, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	filtered := st.RecentSessionIDs[:0]
	for _, id := range st.RecentSessionIDs {
		if id != sessionID {
			filtered = append(filtered, id)
		}
	}
	st.RecentSessionIDs = filtered
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

func migrateRecentsFromSessions(st *store.State, sessions []model.Session) {
	if len(st.RecentSessionIDs) > 0 || len(st.RecentAgents) > 0 {
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
		if len(st.RecentSessionIDs) < maxRecentSessions {
			st.RecentSessionIDs = append(st.RecentSessionIDs, s.ID)
		}
		agentType := strings.TrimSpace(s.AgentType)
		if agentType == "" {
			continue
		}
		if _, ok := seenAgents[agentType]; ok {
			continue
		}
		seenAgents[agentType] = struct{}{}
		if len(st.RecentAgents) >= maxRecentAgents {
			continue
		}
		st.RecentAgents = append(st.RecentAgents, store.RecentAgentEntry{
			AgentType:   agentType,
			WorkingDir:  s.WorkingDir,
			AgentConfig: cloneAgentConfig(s.AgentConfig),
			UsedAt:      sessionActivityTime(s),
		})
	}
}

func recordSessionRecents(s model.Session, st store.State) store.State {
	touchRecentSession(&st, s.ID)
	touchRecentAgent(&st, s.AgentType, s.WorkingDir, s.AgentConfig)
	return st
}

func ensureRecentsPopulated(st store.State) (store.State, bool) {
	if len(st.RecentSessionIDs) > 0 || len(st.RecentAgents) > 0 {
		return st, false
	}
	mu.RLock()
	sessionSnapshot := append([]model.Session(nil), sessions...)
	mu.RUnlock()
	beforeSessions := len(st.RecentSessionIDs)
	beforeAgents := len(st.RecentAgents)
	migrateRecentsFromSessions(&st, sessionSnapshot)
	return st, len(st.RecentSessionIDs) != beforeSessions || len(st.RecentAgents) != beforeAgents
}

func pruneRecentSessionID(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	st, err := store.LoadState()
	if err != nil {
		return
	}
	before := len(st.RecentSessionIDs)
	removeRecentSessionID(&st, sessionID)
	if len(st.RecentSessionIDs) == before {
		return
	}
	if err := store.SaveState(st); err != nil {
		fmt.Fprintf(os.Stderr, "warn: prune recent session id: %v\n", err)
	}
}
