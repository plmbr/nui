// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"nui/internal/model"
	"nui/internal/storageext"
	"nui/internal/store"
	"strings"
)

func sessionUsesExtensionStorage(session model.Session) bool {
	coord := storageext.Default
	return coord != nil && coord.HasSessionHandler(session.AgentType)
}

func persistSessionState(sessionID string, session model.Session, workingDir string) {
	coord := storageext.Default
	if coord == nil || !coord.HasSessionHandler(session.AgentType) {
		return
	}
	mu.RLock()
	msgs := append([]model.ChatMessage(nil), sessionMessages[sessionID]...)
	agentSessionID := agentSessions[sessionID]
	mu.RUnlock()
	coord.WriteSession(sessionID, session.AgentType, workingDir, agentSessionID, msgs)
}

func loadExtensionSessionMessages(sessionID string, session model.Session, workingDir string) {
	if !sessionUsesExtensionStorage(session) {
		return
	}
	mu.RLock()
	hasCached := len(sessionMessages[sessionID]) > 0
	mu.RUnlock()
	if hasCached {
		return
	}
	coord := storageext.Default
	if coord == nil {
		return
	}
	msgs, agentSessionID, err := coord.ReadSession(sessionID, session.AgentType, workingDir)
	if err != nil {
		return
	}
	mu.Lock()
	if len(sessionMessages[sessionID]) == 0 {
		sessionMessages[sessionID] = msgs
	}
	if agentSessionID != "" {
		agentSessions[sessionID] = agentSessionID
	}
	mu.Unlock()
}

func deleteExtensionSession(sessionID string, session model.Session, workingDir, agentSessionID string) {
	coord := storageext.Default
	if coord == nil || !coord.HasSessionHandler(session.AgentType) {
		return
	}
	coord.DeleteSession(sessionID, session.AgentType, workingDir, agentSessionID)
}

func snapshotDataFiltered() store.Data {
	ss := make([]model.Session, len(sessions))
	copy(ss, sessions)
	as := make(map[string]string)
	sm := make(map[string][]model.ChatMessage)
	for k, v := range agentSessions {
		as[k] = v
	}
	for k, v := range sessionMessages {
		copied := make([]model.ChatMessage, len(v))
		copy(copied, v)
		sm[k] = copied
	}
	coord := storageext.Default
	if coord != nil {
		extensionSessions := map[string]bool{}
		for _, session := range sessions {
			if coord.HasSessionHandler(session.AgentType) {
				extensionSessions[session.ID] = true
				delete(as, session.ID)
				delete(sm, session.ID)
			}
		}
		for key := range as {
			if extensionSessions[key] {
				continue
			}
			if i := strings.Index(key, "#"); i > 0 && extensionSessions[key[:i]] {
				delete(as, key)
			}
		}
	}
	return store.Data{Sessions: ss, AgentSessions: as, SessionMessages: sm}
}
