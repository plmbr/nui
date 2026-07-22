// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"nui/internal/mcpclient"
	"nui/internal/model"
	"nui/internal/store"
)

type mcpServerConfig struct {
	Command string                  `json:"command"`
	Args    []string                `json:"args"`
	URL     string                  `json:"url,omitempty"`
	Type    string                  `json:"type,omitempty"`
	Headers map[string]string       `json:"headers,omitempty"`
	Auth    *model.ADLMCPServerAuth `json:"auth,omitempty"`
}

type mcpConfigFile struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

type MCPManager struct {
	mu       sync.RWMutex
	loadOnce sync.Once
	loadErr  error
	client   *mcpclient.Client
}

var mcpManager MCPManager

func isValidMCPServerCfg(cfg mcpServerConfig) bool {
	if u := strings.TrimSpace(cfg.URL); u != "" {
		return true
	}
	cmd := strings.TrimSpace(cfg.Command)
	if cmd == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(cmd))
	if (base == "python3" || base == "python" || base == "node") && len(cfg.Args) == 0 {
		return false
	}
	return true
}

func (cfg mcpServerConfig) toADL(name string) model.ADLMCPServer {
	return model.ADLMCPServer{
		Name:    name,
		URL:     strings.TrimSpace(cfg.URL),
		Command: strings.TrimSpace(cfg.Command),
		Args:    cfg.Args,
		Type:    strings.TrimSpace(cfg.Type),
		Headers: cfg.Headers,
		Auth:    cfg.Auth,
	}
}

func (m *MCPManager) ensureLoaded() error {
	m.loadOnce.Do(func() {
		m.loadErr = bootstrapMCPLoad(m)
	})
	return m.loadErr
}

func (m *MCPManager) load(path string) error {
	cfg, err := readMCPManagerConfig(path)
	if err != nil {
		return err
	}
	if len(cfg.MCPServers) == 0 {
		return nil
	}

	var servers []model.ADLMCPServer
	var skippedInvalid int
	for name, serverCfg := range cfg.MCPServers {
		if !isValidMCPServerCfg(serverCfg) {
			skippedInvalid++
			continue
		}
		servers = append(servers, serverCfg.toADL(name))
	}

	client := mcpclient.New()
	failures := client.ConnectServers(context.Background(), servers)
	for _, msg := range failures {
		fmt.Fprintf(os.Stderr, "[MCP] %s\n", msg)
	}

	m.mu.Lock()
	if m.client != nil {
		m.client.Close()
	}
	m.client = client
	m.mu.Unlock()

	if skippedInvalid > 0 {
		fmt.Fprintf(os.Stderr, "[MCP] skipped %d server(s) with invalid or incomplete config\n", skippedInvalid)
	}

	return nil
}

func readMCPManagerConfig(path string) (mcpConfigFile, error) {
	var cfg mcpConfigFile
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			_ = json.Unmarshal(data, &cfg)
		}
	}
	if cfg.MCPServers == nil {
		cfg.MCPServers = map[string]mcpServerConfig{}
	}
	userServers, err := store.LoadMCPServers()
	if err != nil {
		return cfg, err
	}
	for _, srv := range userServers {
		name := strings.TrimSpace(srv.Name)
		if name == "" {
			continue
		}
		if _, exists := cfg.MCPServers[name]; exists {
			continue
		}
		if strings.TrimSpace(srv.URL) == "" {
			continue
		}
		cfg.MCPServers[name] = mcpServerConfig{
			URL:     srv.URL,
			Type:    srv.Type,
			Headers: srv.Headers,
			Auth:    srv.Auth,
		}
	}
	return cfg, nil
}

func (m *MCPManager) clientOrNil() *mcpclient.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client
}

func (m *MCPManager) readResource(ctx context.Context, serverName, uri string) (string, error) {
	if err := m.ensureLoaded(); err != nil {
		return "", err
	}
	client := m.clientOrNil()
	if client == nil {
		return "", fmt.Errorf("MCP server %q not found (configure ~/.nui/.mcp.json)", serverName)
	}
	return client.ReadResource(ctx, serverName, uri)
}

func (m *MCPManager) callTool(ctx context.Context, serverName, name string, args map[string]any) (any, error) {
	if err := m.ensureLoaded(); err != nil {
		return nil, err
	}
	client := m.clientOrNil()
	if client == nil {
		return nil, fmt.Errorf("MCP server %q not found (configure ~/.nui/.mcp.json)", serverName)
	}
	return client.CallToolStructured(ctx, serverName, name, args)
}

func (m *MCPManager) LookupToolUI(toolName string) (resourceURI, serverName string, ok bool) {
	if err := m.ensureLoaded(); err != nil {
		return "", "", false
	}
	client := m.clientOrNil()
	if client == nil {
		return "", "", false
	}
	return client.LookupToolUI(toolName)
}

// LookupSessionToolUI checks a session-scoped MCP client first, then the global manager.
func LookupSessionToolUI(sessionID, toolName string) (resourceURI, serverName string, ok bool) {
	if sessionID != "" && extensionManager != nil {
		if client := extensionManager.SessionMCPClient(sessionID); client != nil {
			if uri, server, found := client.LookupToolUI(toolName); found {
				return uri, server, true
			}
		}
	}
	return mcpManager.LookupToolUI(toolName)
}

// ReadMCPResource reads a resource from a session client or the global manager.
func ReadMCPResource(ctx context.Context, sessionID, serverName, uri string) (string, error) {
	if sessionID != "" && extensionManager != nil {
		if client := extensionManager.SessionMCPClient(sessionID); client != nil {
			if html, err := client.ReadResource(ctx, serverName, uri); err == nil {
				return html, nil
			}
		}
	}
	return mcpManager.readResource(ctx, serverName, uri)
}
