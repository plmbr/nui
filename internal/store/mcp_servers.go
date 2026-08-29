// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"nui/internal/model"
)

func MCPServersPath() (string, error) {
	dir, err := UserDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mcp-servers.json"), nil
}

func loadMCPServersFile(path string) ([]model.ADLMCPServer, error) {
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

func mcpServerKey(s model.ADLMCPServer) string {
	return strings.TrimSpace(s.Name)
}

// LoadUserMCPServers reads MCP servers from the user data dir only.
func LoadUserMCPServers() ([]model.ADLMCPServer, error) {
	path, err := MCPServersPath()
	if err != nil {
		return nil, err
	}
	return loadMCPServersFile(path)
}

// LoadMCPServers returns system + user MCP servers (user wins on same name/id).
func LoadMCPServers() ([]model.ADLMCPServer, error) {
	byKey := map[string]model.ADLMCPServer{}
	order := []string{}

	add := func(servers []model.ADLMCPServer) {
		for _, s := range servers {
			key := mcpServerKey(s)
			if key == "" {
				continue
			}
			if _, exists := byKey[key]; !exists {
				order = append(order, key)
			}
			byKey[key] = s
		}
	}

	if SystemDirExists() {
		sys, err := loadMCPServersFile(filepath.Join(SystemDir(), "mcp-servers.json"))
		if err == nil {
			add(sys)
		}
	}
	user, err := LoadUserMCPServers()
	if err != nil {
		return nil, err
	}
	add(user)

	out := make([]model.ADLMCPServer, 0, len(order))
	for _, key := range order {
		out = append(out, byKey[key])
	}
	return out, nil
}

// SaveMCPServers writes user MCP server definitions to the user data dir.
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
