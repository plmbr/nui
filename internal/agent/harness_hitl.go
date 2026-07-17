// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"strings"

	"nui/internal/model"
)

const (
	EnvNuiSessionID = "NUI_SESSION_ID"
	EnvnuiRunID     = "NUI_RUN_ID"
	EnvnuiAPIURL    = "NUI_API_URL"
	nuiHitlMCPName  = "nui-hitl"
)

func defaultnuiAPIURL() string {
	if v := strings.TrimSpace(os.Getenv("NUI_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8080"
}

func nuiExecutable() (string, error) {
	if path := strings.TrimSpace(os.Getenv("NUI_MCP_BINARY")); path != "" {
		return path, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// Prefer the running binary (including go run's go-build output) so dev builds
	// always expose the latest MCP subcommands.
	return exe, nil
}

func nuiHitlMCPServer(sessionID, apiURL string) (model.ADLMCPServer, error) {
	exe, err := nuiExecutable()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	if apiURL == "" {
		apiURL = defaultnuiAPIURL()
	}
	return model.ADLMCPServer{
		Name:    nuiHitlMCPName,
		Command: exe,
		Args:    []string{"hitl-mcp"},
		Env: map[string]string{
			EnvNuiSessionID: sessionID,
			EnvnuiAPIURL:    apiURL,
		},
	}, nil
}

func hasnuiHitlMCP(servers []model.ADLMCPServer) bool {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == nuiHitlMCPName {
			return true
		}
	}
	return false
}

func appendNuiHitlMCP(servers []model.ADLMCPServer, sessionID, apiURL string) ([]model.ADLMCPServer, error) {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == nuiHitlMCPName {
			return servers, nil
		}
	}
	srv, err := nuiHitlMCPServer(sessionID, apiURL)
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}

func nuiSessionIDForRun(req RunRequest, projectID string) string {
	if req.NuiSessionID != "" {
		return req.NuiSessionID
	}
	return projectID
}

// harnessResumeSessionID returns the harness-native session id when it is distinct from the nui session id.
func harnessResumeSessionID(req RunRequest) string {
	if req.SessionID == "" {
		return ""
	}
	if req.NuiSessionID != "" && req.SessionID == req.NuiSessionID {
		return ""
	}
	return req.SessionID
}

func nuiHarnessEnv(sessionID, runID, apiURL string) map[string]string {
	if apiURL == "" {
		apiURL = defaultnuiAPIURL()
	}
	env := map[string]string{
		EnvnuiAPIURL: apiURL,
	}
	if sessionID != "" {
		env[EnvNuiSessionID] = sessionID
	}
	if runID != "" {
		env[EnvnuiRunID] = runID
	}
	return env
}

func mergenuiHarnessEnv(base map[string]string, sessionID, runID, apiURL string) map[string]string {
	out := make(map[string]string, len(base)+3)
	for k, v := range base {
		out[k] = v
	}
	for k, v := range nuiHarnessEnv(sessionID, runID, apiURL) {
		out[k] = v
	}
	return out
}

const hitlSystemPromptAppendix = `
## Human in the loop (nui HITL)

When you need input, preferences, or clarifications from the human, call the **ask_user** tool on the **nui-hitl** MCP server. Do not ask those questions only in assistant text—the human answers through the nui UI prompt card.

Each question's options must be objects with a **label** field (and optional **description**), for example: {"label": "Red", "description": "Bright red"}. Do not pass bare strings in the options array.

For approve/reject gates before risky actions, use **request_approval** on **nui-hitl**.
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
