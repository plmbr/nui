// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"nui/internal/memory"
	"nui/internal/model"
	"nui/internal/store"
)

const nuiAgentMCPName = "nui-agent"

func nuiAgentMCPServer(agentID string) (model.ADLMCPServer, error) {
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
	srv, err := nuiAgentMCPServer(agentID)
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}

const builtinNuiMCPRef = "builtin:nui"
const nuiGeneralMCPName = "nui"

// expandBuiltinNuiMCPRefs resolves ref: builtin:nui to the general nui MCP (nui mcp).
func expandBuiltinNuiMCPRefs(servers []model.ADLMCPServer) []model.ADLMCPServer {
	if len(servers) == 0 {
		return servers
	}
	out := make([]model.ADLMCPServer, 0, len(servers))
	for _, s := range servers {
		if strings.TrimSpace(s.Ref) != builtinNuiMCPRef {
			out = append(out, s)
			continue
		}
		exe, err := nuiExecutable()
		if err != nil {
			out = append(out, s)
			continue
		}
		name := strings.TrimSpace(s.Name)
		if name == "" {
			name = nuiGeneralMCPName
		}
		env := map[string]string{}
		for k, v := range s.Env {
			env[k] = v
		}
		if _, ok := env[EnvnuiAPIURL]; !ok {
			env[EnvnuiAPIURL] = defaultnuiAPIURL()
		}
		// Also accept NUI_URL as used by the council workaround.
		if _, ok := env["NUI_URL"]; !ok {
			env["NUI_URL"] = defaultnuiAPIURL()
		}
		out = append(out, model.ADLMCPServer{
			Name:    name,
			Command: exe,
			Args:    []string{"mcp"},
			Type:    "stdio",
			Env:     env,
		})
	}
	return out
}
