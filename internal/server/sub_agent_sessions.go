// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"os"

	"loop/internal/agent"
	"loop/internal/agents"
	"loop/internal/model"
	"loop/internal/store"
)

func subAgentSessionKey(sessionID, subAgentID string) string {
	return fmt.Sprintf("%s#%s", sessionID, subAgentID)
}

func getSubAgentSessionID(sessionID, subAgentID string) string {
	mu.RLock()
	defer mu.RUnlock()
	return agentSessions[subAgentSessionKey(sessionID, subAgentID)]
}

func setSubAgentSessionID(sessionID, subAgentID, harnessSessionID string) {
	if harnessSessionID == "" {
		return
	}
	key := subAgentSessionKey(sessionID, subAgentID)
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
		fmt.Fprintf(os.Stderr, "warn: save sub-agent session: %v\n", err)
	}
}

func wireOrchestratorRunRequest(runReq *agent.RunRequest, sessionID string, def model.ADLDefinition) {
	if !model.IsOrchestratorAgent(def) {
		return
	}
	runReq.ResolveADL = agents.LookupDefinition
	runReq.SubAgentHarnessSession = func(subAgentID string) string {
		return getSubAgentSessionID(sessionID, subAgentID)
	}
	runReq.OnSubAgentHarnessSession = func(subAgentID, harnessSessionID string) {
		setSubAgentSessionID(sessionID, subAgentID, harnessSessionID)
	}
}
