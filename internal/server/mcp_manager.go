// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"nui/internal/mcpoauth"
	"nui/internal/model"
	"nui/internal/store"
)

type mcpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	URL     string            `json:"url,omitempty"`
	Type    string            `json:"type,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Auth    *model.ADLMCPServerAuth `json:"auth,omitempty"`
}

type mcpConfigFile struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

type mcpSession struct {
	name    string
	session *mcp.ClientSession
}

type MCPManager struct {
	mu         sync.RWMutex
	loadOnce   sync.Once
	loadErr    error
	sessions   map[string]*mcpSession
	toolUI     map[string]string
	toolServer map[string]string
}

var mcpManager MCPManager

const mcpConnectTimeout = 15 * time.Second

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

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sessions == nil {
		m.sessions = map[string]*mcpSession{}
	}
	if m.toolUI == nil {
		m.toolUI = map[string]string{}
	}
	if m.toolServer == nil {
		m.toolServer = map[string]string{}
	}

	var skippedInvalid int
	for name, serverCfg := range cfg.MCPServers {
		if !isValidMCPServerCfg(serverCfg) {
			skippedInvalid++
			continue
		}
		srv := serverCfg.toADL(name)
		ctx, cancel := context.WithTimeout(context.Background(), mcpConnectTimeout)
		var session *mcp.ClientSession
		var connectErr error
		if strings.TrimSpace(srv.URL) != "" {
			session, connectErr = mcpoauth.ConnectRemote(ctx, srv)
		} else {
			// Use exec.Command (not CommandContext): the connect timeout context must not
			// be tied to the child process lifetime or cancel() kills stdio MCP servers
			// right after Connect returns, breaking later tools/call requests.
			cmd := exec.Command(serverCfg.Command, serverCfg.Args...)
			transport := &mcp.CommandTransport{Command: cmd}
			client := mcp.NewClient(&mcp.Implementation{Name: "nui", Version: "1.0.0"}, nil)
			session, connectErr = client.Connect(ctx, transport, nil)
		}
		cancel()
		if connectErr != nil {
			fmt.Fprintf(os.Stderr, "[MCP] failed to connect to %q: %v\n", name, connectErr)
			continue
		}

		listCtx, listCancel := context.WithTimeout(context.Background(), mcpConnectTimeout)
		tools, err := session.ListTools(listCtx, &mcp.ListToolsParams{})
		listCancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[MCP] list tools for %q: %v\n", name, err)
			_ = session.Close()
			continue
		}
		for _, tool := range tools.Tools {
			if uri := toolUIResourceURI(tool); uri != "" {
				m.toolUI[tool.Name] = uri
				m.toolServer[tool.Name] = name
			}
		}

		m.sessions[name] = &mcpSession{name: name, session: session}
		fmt.Fprintf(os.Stderr, "[MCP] connected to %q\n", name)
	}

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

func toolUIResourceURI(tool *mcp.Tool) string {
	if tool == nil || tool.Meta == nil {
		return ""
	}
	meta := tool.Meta
	if ui, ok := meta["ui"].(map[string]any); ok {
		if uri, ok := ui["resourceUri"].(string); ok && strings.HasPrefix(uri, "ui://") {
			return uri
		}
	}
	if uri, ok := meta["ui/resourceUri"].(string); ok && strings.HasPrefix(uri, "ui://") {
		return uri
	}
	return ""
}

func (m *MCPManager) session(name string) (*mcp.ClientSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[name]
	if !ok || s.session == nil {
		return nil, fmt.Errorf("MCP server %q not found (configure ~/.nui/.mcp.json)", name)
	}
	return s.session, nil
}

func (m *MCPManager) readResource(ctx context.Context, serverName, uri string) (string, error) {
	if err := m.ensureLoaded(); err != nil {
		return "", err
	}
	session, err := m.session(serverName)
	if err != nil {
		return "", err
	}
	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		return "", err
	}
	for _, content := range result.Contents {
		if content.Text != "" {
			return content.Text, nil
		}
	}
	return "", fmt.Errorf("no HTML content in resource")
}

func (m *MCPManager) callTool(ctx context.Context, serverName, name string, args map[string]any) (any, error) {
	if err := m.ensureLoaded(); err != nil {
		return nil, err
	}
	session, err := m.session(serverName)
	if err != nil {
		return nil, err
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}
	return mcpCallToolResult(result), nil
}

func mcpCallToolResult(result *mcp.CallToolResult) map[string]any {
	out := map[string]any{
		"isError": result.IsError,
	}
	if result.StructuredContent != nil {
		out["structuredContent"] = result.StructuredContent
	}
	var content []map[string]any
	for _, c := range result.Content {
		switch v := c.(type) {
		case *mcp.TextContent:
			content = append(content, map[string]any{"type": "text", "text": v.Text})
		default:
			content = append(content, map[string]any{"type": "text", "text": fmt.Sprintf("%v", v)})
		}
	}
	out["content"] = content
	return out
}

func (m *MCPManager) LookupToolUI(toolName string) (resourceURI, serverName string, ok bool) {
	if err := m.ensureLoaded(); err != nil {
		return "", "", false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if uri, found := m.toolUI[toolName]; found {
		return uri, m.toolServer[toolName], true
	}
	for _, sep := range []string{"__", ":", "_"} {
		if idx := strings.LastIndex(toolName, sep); idx >= 0 {
			bare := toolName[idx+len(sep):]
			if uri, found := m.toolUI[bare]; found {
				return uri, m.toolServer[bare], true
			}
		}
	}
	return "", "", false
}
