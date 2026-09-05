// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"nui/internal/agent"
	"nui/internal/agents"
	"nui/internal/model"
	"nui/internal/store"
)

const (
	agentConfigCouncilParent = "councilParentSessionId"
	agentConfigCouncilMember = "councilMemberId"
)

func memberSessionKey(sessionID, memberID string) string {
	return fmt.Sprintf("%s#%s", sessionID, memberID)
}

func getMemberSessionID(sessionID, memberID string) string {
	mu.RLock()
	defer mu.RUnlock()
	return agentSessions[memberSessionKey(sessionID, memberID)]
}

func setMemberSessionID(sessionID, memberID, harnessSessionID string) {
	if harnessSessionID == "" {
		return
	}
	key := memberSessionKey(sessionID, memberID)
	mu.Lock()
	agentSessions[key] = harnessSessionID
	var skipSave bool
	if s, ok := findSession(sessionID); ok {
		skipSave = sessionUsesExtensionStorage(s)
	}
	snapshot := snapshotData()
	mu.Unlock()
	if skipSave {
		return
	}
	if err := store.SaveData(snapshot); err != nil {
		fmt.Fprintf(os.Stderr, "warn: save council member session: %v\n", err)
	}
}

func isCouncilManagedSession(s model.Session) bool {
	if s.AgentConfig == nil {
		return false
	}
	parent, _ := s.AgentConfig[agentConfigCouncilParent].(string)
	return strings.TrimSpace(parent) != ""
}

func councilParentSessionID(s model.Session) string {
	if s.AgentConfig == nil {
		return ""
	}
	parent, _ := s.AgentConfig[agentConfigCouncilParent].(string)
	return strings.TrimSpace(parent)
}

func councilMemberIDFromSession(s model.Session) string {
	if s.AgentConfig == nil {
		return ""
	}
	id, _ := s.AgentConfig[agentConfigCouncilMember].(string)
	return strings.TrimSpace(id)
}

func filterPublicSessions(list []model.Session) []model.Session {
	out := make([]model.Session, 0, len(list))
	for _, s := range list {
		if isCouncilManagedSession(s) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func findCouncilChildSession(parentSessionID, memberID string) (model.Session, bool) {
	mu.RLock()
	defer mu.RUnlock()
	for _, s := range sessions {
		if councilParentSessionID(s) == parentSessionID && councilMemberIDFromSession(s) == memberID {
			return s, true
		}
	}
	return model.Session{}, false
}

func listCouncilChildSessionIDs(parentSessionID string) []string {
	mu.RLock()
	defer mu.RUnlock()
	var ids []string
	for _, s := range sessions {
		if councilParentSessionID(s) == parentSessionID {
			ids = append(ids, s.ID)
		}
	}
	return ids
}

func deleteCouncilChildSessions(parentSessionID string) {
	ids := listCouncilChildSessionIDs(parentSessionID)
	for _, id := range ids {
		info, ok := sessionDeleteInfoFor(id)
		if !ok {
			continue
		}
		mu.Lock()
		purgeSessionFromMemory(id)
		snapshot := snapshotData()
		mu.Unlock()
		_ = store.SaveData(snapshot)
		cleanupDeletedSession(id, info)
	}
}

func ensureCouncilMemberChildSession(parent model.Session, memberID, label, agentType string, fresh bool) (string, error) {
	memberID = strings.TrimSpace(memberID)
	agentType = strings.TrimSpace(agentType)
	if memberID == "" || agentType == "" {
		return "", fmt.Errorf("council member id and agent type required")
	}
	if !fresh {
		if existing, ok := findCouncilChildSession(parent.ID, memberID); ok {
			return existing.ID, nil
		}
	} else if existing, ok := findCouncilChildSession(parent.ID, memberID); ok {
		info, _ := sessionDeleteInfoFor(existing.ID)
		mu.Lock()
		purgeSessionFromMemory(existing.ID)
		snapshot := snapshotData()
		mu.Unlock()
		_ = store.SaveData(snapshot)
		cleanupDeletedSession(existing.ID, info)
	}

	cfg := cloneAgentConfig(parent.AgentConfig)
	if cfg == nil {
		cfg = map[string]any{}
	}
	cfg[agentConfigCouncilParent] = parent.ID
	cfg[agentConfigCouncilMember] = memberID
	name := strings.TrimSpace(label)
	if name == "" {
		name = agentType
	}
	name = "Council · " + name

	s, err := createSession(name, parent.WorkingDir, agentType, cfg)
	if err != nil {
		return "", err
	}
	return s.ID, nil
}

func runCouncilMemberChildSession(ctx context.Context, childSessionID, message string, onStarted func(runID string)) (output, runID string, err error) {
	mu.RLock()
	session, ok := findSession(childSessionID)
	agentSessionID := agentSessions[childSessionID]
	mu.RUnlock()
	if !ok {
		return "", "", fmt.Errorf("council child session %q not found", childSessionID)
	}

	runID = uuid.NewString()
	createRunRecord(childSessionID, runID, message)
	appendUserMessage(childSessionID, message)
	if onStarted != nil {
		onStarted(runID)
	}

	runCtx, cancel := context.WithCancel(ctx)
	registerActiveRun(childSessionID, runID, cancel)
	defer unregisterActiveRun(childSessionID, runID)

	result := executeRun(runCtx, executeRunOptions{
		SessionID:       childSessionID,
		Session:         session,
		Message:         message,
		ResolveMentions: false,
		RunID:           runID,
		PersistLog:      true,
		AgentSessionID:  agentSessionID,
	})

	var status RunStatus
	var errMsg string
	switch {
	case result.Cancelled:
		status = RunStatusCancelled
		errMsg = "cancelled"
	case result.Errored:
		status = RunStatusFailed
		errMsg = result.ErrorMessage
	default:
		status = RunStatusCompleted
	}
	persistAssistantTurn(childSessionID, result.AssistantContent, result.NewAgentSessionID, sessionSkipsTopLevelHarnessSession(session))
	finishRunRecord(runID, status, result.AssistantContent, errMsg)
	scheduleMaybeGenerateSessionTitle(childSessionID)

	out := strings.TrimSpace(result.AssistantContent)
	if result.Cancelled {
		return out, runID, ctx.Err()
	}
	if result.Errored {
		if errMsg == "" {
			errMsg = "member run failed"
		}
		return out, runID, fmt.Errorf("%s", errMsg)
	}
	if out == "" {
		return "", runID, fmt.Errorf("empty output")
	}
	return out, runID, nil
}

func wireOrchestratorRunRequest(runReq *agent.RunRequest, sessionID string, def model.ADLDefinition) {
	runReq.ResolveADL = agents.LookupDefinition
	if !model.IsCouncilAgent(def) && !model.IsSubAgentsOrchestration(def) {
		return
	}
	runReq.MemberHarnessSession = func(memberID string) string {
		return getMemberSessionID(sessionID, memberID)
	}
	runReq.OnMemberHarnessSession = func(memberID, harnessSessionID string) {
		setMemberSessionID(sessionID, memberID, harnessSessionID)
	}

	sessionMode := "persistent"
	if def.Orchestration != nil && strings.TrimSpace(def.Orchestration.SessionMode) != "" {
		sessionMode = strings.TrimSpace(def.Orchestration.SessionMode)
	}
	fresh := sessionMode == "fresh"

	runReq.EnsureCouncilMemberSession = func(memberID, label, agentType string) (string, error) {
		mu.RLock()
		parent, ok := findSession(sessionID)
		mu.RUnlock()
		if !ok {
			return "", fmt.Errorf("parent session %q not found", sessionID)
		}
		return ensureCouncilMemberChildSession(parent, memberID, label, agentType, fresh)
	}
	runReq.RunCouncilMemberSession = func(ctx context.Context, childSessionID, message string, onStarted func(runID string)) (string, string, error) {
		return runCouncilMemberChildSession(ctx, childSessionID, message, onStarted)
	}
}
