// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"loop/internal/model"
)

func MCPServersPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mcp-servers.json"), nil
}

// LoadMCPServers reads user MCP server definitions from ~/.loop/mcp-servers.json.
func LoadMCPServers() ([]model.ADLMCPServer, error) {
	path, err := MCPServersPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []model.ADLMCPServer{}, nil
	}
	if err != nil {
		return nil, err
	}
	var wrap struct {
		MCPServers []model.ADLMCPServer `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return nil, err
	}
	if wrap.MCPServers == nil {
		return []model.ADLMCPServer{}, nil
	}
	return wrap.MCPServers, nil
}

// SaveMCPServers writes user MCP server definitions to ~/.loop/mcp-servers.json.
func SaveMCPServers(servers []model.ADLMCPServer) error {
	path, err := MCPServersPath()
	if err != nil {
		return err
	}
	if servers == nil {
		servers = []model.ADLMCPServer{}
	}
	wrap := struct {
		MCPServers []model.ADLMCPServer `json:"mcpServers"`
	}{MCPServers: servers}
	data, err := json.MarshalIndent(wrap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
