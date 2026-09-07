// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"nui/internal/model"
)

const nuiBashMCPName = "nui-bash"

func nuiBashMCPServer(workingDir string) (model.ADLMCPServer, error) {
	exe, err := nuiExecutable()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	env := map[string]string{}
	if wd := strings.TrimSpace(workingDir); wd != "" {
		env[envNuiWorkingDir] = wd
	}
	return model.ADLMCPServer{
		Name:    nuiBashMCPName,
		Command: exe,
		Args:    []string{"bash-mcp"},
		Env:     env,
	}, nil
}

func hasNuiBashMCP(servers []model.ADLMCPServer) bool {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == nuiBashMCPName {
			return true
		}
	}
	return false
}

func appendNuiBashMCP(servers []model.ADLMCPServer, workingDir string) ([]model.ADLMCPServer, error) {
	if hasNuiBashMCP(servers) {
		return servers, nil
	}
	srv, err := nuiBashMCPServer(workingDir)
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}
