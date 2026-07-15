// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"loop/internal/model"
)

const loopAgentMCPName = "loop-agent"

func loopAgentMCPServer() (model.ADLMCPServer, error) {
	exe, err := loopExecutable()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	return model.ADLMCPServer{
		Name:    loopAgentMCPName,
		Command: exe,
		Args:    []string{"agent-mcp"},
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

func appendLoopAgentMCP(servers []model.ADLMCPServer) ([]model.ADLMCPServer, error) {
	if hasLoopAgentMCP(servers) {
		return servers, nil
	}
	srv, err := loopAgentMCPServer()
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}
