// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"nui/internal/model"
)

const nuiOrchestratorMCPName = "nui-orchestrator"

// NuiOrchestratorMCPServers returns MCP servers for the internal launcher orchestrator.
func NuiOrchestratorMCPServers(apiURL string) ([]model.ADLMCPServer, error) {
	exe, err := nuiExecutable()
	if err != nil {
		return nil, err
	}
	if apiURL == "" {
		apiURL = defaultnuiAPIURL()
	}
	return []model.ADLMCPServer{{
		Name:    nuiOrchestratorMCPName,
		Command: exe,
		Args:    []string{"orchestrator-mcp"},
		Env: map[string]string{
			EnvnuiAPIURL: apiURL,
		},
	}}, nil
}

func isOrchestratorAgent(def model.ADLDefinition) bool {
	return strings.TrimSpace(def.ID) == "nui"
}
