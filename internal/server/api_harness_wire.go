// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"loop/internal/agent"
	"loop/internal/model"
)

// wireAPIHarnessRunRequest fills api-harness fields on RunRequest from session state.
func wireAPIHarnessRunRequest(runReq *agent.RunRequest, sessionID string, def model.ADLDefinition) {
	if def.Harness.Type != "api" {
		return
	}
	mu.RLock()
	msgs := append([]model.ChatMessage(nil), sessionMessages[sessionID]...)
	mu.RUnlock()
	if len(msgs) > 0 && msgs[len(msgs)-1].Role == "user" {
		msgs = msgs[:len(msgs)-1]
	}
	runReq.History = msgs
}
