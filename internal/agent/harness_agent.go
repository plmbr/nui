// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"nui/internal/memory"
	"nui/internal/model"
	"nui/internal/store"
)

const nuiAgentMCPName = "nui-agent"

func loopAgentMCPServer(agentID string) (model.ADLMCPServer, error) {
	exe, err := nuiExecutable()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	env := map[string]string{}
	if id := strings.TrimSpace(agentID); id != "" {
		env[memory.EnvnuiMemoryAgentID] = id
	}
	if settings, err := store.LoadSettings(); err == nil {
		env[memory.EnvnuiMemoryUserMode] = memory.UserMode(settings)
		env[memory.EnvnuiMemoryAgentMode] = memory.AgentMode(settings, agentID)
	}
	return model.ADLMCPServer{
		Name:    nuiAgentMCPName,
		Command: exe,
		Args:    []string{"agent-mcp"},
		Env:     env,
	}, nil
}

func hasNuiAgentMCP(servers []model.ADLMCPServer) bool {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == nuiAgentMCPName {
			return true
		}
	}
	return false
}

func appendNuiAgentMCP(servers []model.ADLMCPServer, agentID string) ([]model.ADLMCPServer, error) {
	if hasNuiAgentMCP(servers) {
		return servers, nil
	}
	srv, err := loopAgentMCPServer(agentID)
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}
