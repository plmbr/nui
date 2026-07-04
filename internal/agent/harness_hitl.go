// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"strings"

	"loop/internal/model"
)

const (
	EnvLoopSessionID = "LOOP_SESSION_ID"
	EnvLoopRunID     = "LOOP_RUN_ID"
	EnvLoopAPIURL    = "LOOP_API_URL"
	loopHitlMCPName  = "loop-hitl"
)

func defaultLoopAPIURL() string {
	if v := strings.TrimSpace(os.Getenv("LOOP_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8080"
}

func loopExecutable() (string, error) {
	return os.Executable()
}

func loopHitlMCPServer(sessionID, apiURL string) (model.ADLMCPServer, error) {
	exe, err := loopExecutable()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	if apiURL == "" {
		apiURL = defaultLoopAPIURL()
	}
	return model.ADLMCPServer{
		Name:    loopHitlMCPName,
		Command: exe,
		Args:    []string{"hitl-mcp"},
		Env: map[string]string{
			EnvLoopSessionID: sessionID,
			EnvLoopAPIURL:    apiURL,
		},
	}, nil
}

func hasLoopHitlMCP(servers []model.ADLMCPServer) bool {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == loopHitlMCPName {
			return true
		}
	}
	return false
}

func appendLoopHitlMCP(servers []model.ADLMCPServer, sessionID, apiURL string) ([]model.ADLMCPServer, error) {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == loopHitlMCPName {
			return servers, nil
		}
	}
	srv, err := loopHitlMCPServer(sessionID, apiURL)
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}

func loopSessionIDForRun(req RunRequest, projectID string) string {
	if req.LoopSessionID != "" {
		return req.LoopSessionID
	}
	return projectID
}

// harnessResumeSessionID returns the harness-native session id when it is distinct from the Loop session id.
func harnessResumeSessionID(req RunRequest) string {
	if req.SessionID == "" {
		return ""
	}
	if req.LoopSessionID != "" && req.SessionID == req.LoopSessionID {
		return ""
	}
	return req.SessionID
}

func loopHarnessEnv(sessionID, runID, apiURL string) map[string]string {
	if apiURL == "" {
		apiURL = defaultLoopAPIURL()
	}
	env := map[string]string{
		EnvLoopAPIURL: apiURL,
	}
	if sessionID != "" {
		env[EnvLoopSessionID] = sessionID
	}
	if runID != "" {
		env[EnvLoopRunID] = runID
	}
	return env
}

func mergeLoopHarnessEnv(base map[string]string, sessionID, runID, apiURL string) map[string]string {
	out := make(map[string]string, len(base)+3)
	for k, v := range base {
		out[k] = v
	}
	for k, v := range loopHarnessEnv(sessionID, runID, apiURL) {
		out[k] = v
	}
	return out
}

const hitlSystemPromptAppendix = `
## Human in the loop (Loop HITL)

When you need input, preferences, or clarifications from the human, call the **ask_user** tool on the **loop-hitl** MCP server. Do not ask those questions only in assistant text—the human answers through the Loop UI prompt card.

Each question's options must be objects with a **label** field (and optional **description**), for example: {"label": "Red", "description": "Bright red"}. Do not pass bare strings in the options array.

For approve/reject gates before risky actions, use **request_approval** on **loop-hitl**.
`

func appendHitlSystemPrompt(systemPrompt string) string {
	block := strings.TrimSpace(hitlSystemPromptAppendix)
	if block == "" {
		return systemPrompt
	}
	base := strings.TrimSpace(systemPrompt)
	if base == "" {
		return block
	}
	return base + "\n\n" + block
}
