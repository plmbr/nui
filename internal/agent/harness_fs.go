// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"nui/internal/model"
)

const nuiFSMCPName = "nui-fs"

func nuiFSMCPServer(workingDir string) (model.ADLMCPServer, error) {
	exe, err := nuiExecutable()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	env := map[string]string{}
	if wd := strings.TrimSpace(workingDir); wd != "" {
		env[envNuiWorkingDir] = wd
	}
	return model.ADLMCPServer{
		Name:    nuiFSMCPName,
		Command: exe,
		Args:    []string{"fs-mcp"},
		Env:     env,
	}, nil
}

func hasNuiFSMCP(servers []model.ADLMCPServer) bool {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == nuiFSMCPName {
			return true
		}
	}
	return false
}

func appendNuiFSMCP(servers []model.ADLMCPServer, workingDir string) ([]model.ADLMCPServer, error) {
	if hasNuiFSMCP(servers) {
		return servers, nil
	}
	srv, err := nuiFSMCPServer(workingDir)
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}
