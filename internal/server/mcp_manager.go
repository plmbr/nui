// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
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
	sessions   map[string]*mcpSession
	toolUI     map[string]string
	toolServer map[string]string
}

var mcpManager MCPManager

func (m *MCPManager) load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg mcpConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
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

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "loop", Version: "1.0.0"}, nil)

	for name, serverCfg := range cfg.MCPServers {
		if serverCfg.Command == "" {
			continue
		}
		cmd := exec.Command(serverCfg.Command, serverCfg.Args...)
		transport := &mcp.CommandTransport{Command: cmd}
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[MCP] failed to connect to %q: %v\n", name, err)
			continue
		}

		tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "[MCP] list tools for %q: %v\n", name, err)
		} else {
			for _, tool := range tools.Tools {
				if uri := toolUIResourceURI(tool); uri != "" {
					m.toolUI[tool.Name] = uri
					m.toolServer[tool.Name] = name
				}
			}
		}

		m.sessions[name] = &mcpSession{name: name, session: session}
		fmt.Fprintf(os.Stderr, "[MCP] connected to %q\n", name)
	}

	return nil
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
		return nil, fmt.Errorf("MCP server %q not found (configure ~/.loop/.mcp.json)", name)
	}
	return s.session, nil
}

func (m *MCPManager) readResource(ctx context.Context, serverName, uri string) (string, error) {
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
