// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"loop/internal/memory"
	"loop/internal/model"
	"loop/internal/store"
)

const loopAgentMCPName = "loop-agent"

func loopAgentMCPServer(agentID string) (model.ADLMCPServer, error) {
	exe, err := loopExecutable()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	env := map[string]string{}
	if id := strings.TrimSpace(agentID); id != "" {
		env[memory.EnvLoopMemoryAgentID] = id
	}
	if settings, err := store.LoadSettings(); err == nil {
		env[memory.EnvLoopMemoryUserMode] = memory.UserMode(settings)
		env[memory.EnvLoopMemoryAgentMode] = memory.AgentMode(settings, agentID)
	}
	return model.ADLMCPServer{
		Name:    loopAgentMCPName,
		Command: exe,
		Args:    []string{"agent-mcp"},
		Env:     env,
	}, nil
}

func hasLoopAgentMCP(servers []model.ADLMCPServer) bool {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == loopAgentMCPName {
			return true
		}
	}
	return false
}

func appendLoopAgentMCP(servers []model.ADLMCPServer, agentID string) ([]model.ADLMCPServer, error) {
	if hasLoopAgentMCP(servers) {
		return servers, nil
	}
	srv, err := loopAgentMCPServer(agentID)
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}
