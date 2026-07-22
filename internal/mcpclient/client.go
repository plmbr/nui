// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"nui/internal/mcpoauth"
	"nui/internal/model"
)

const defaultConnectTimeout = 15 * time.Second

// Tool describes an MCP tool exposed to LLM callers.
type Tool struct {
	Name        string
	Description string
	Server      string
	InputSchema map[string]any
}

type toolEntry struct {
	displayName string
	bareName    string
	server      string
	description string
	inputSchema map[string]any
}

// Client connects to MCP servers and exposes tools, resources, and MCP App UI metadata.
type Client struct {
	mu         sync.RWMutex
	sessions   map[string]*mcp.ClientSession
	tools      []Tool
	toolMap    map[string]toolEntry
	toolUI     map[string]string
	toolServer map[string]string
	logPrefix  string
}

// New creates an MCP client. Call ConnectServers before use.
func New() *Client {
	return &Client{
		sessions:   map[string]*mcp.ClientSession{},
		toolMap:    map[string]toolEntry{},
		toolUI:     map[string]string{},
		toolServer: map[string]string{},
		logPrefix:  "[mcpclient]",
	}
}

// ConnectServers connects to the given MCP server list and returns user-facing
// messages for servers that failed to connect.
func (c *Client) ConnectServers(ctx context.Context, servers []model.ADLMCPServer) []string {
	type listedTool struct {
		server      string
		bareName    string
		description string
		inputSchema map[string]any
		uiURI       string
	}

	var failures []string
	var listed []listedTool

	for _, srv := range servers {
		name := strings.TrimSpace(srv.Name)
		if name == "" {
			continue
		}
		session, err := c.connectWithTimeout(ctx, srv, defaultConnectTimeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s connect %q: %v\n", c.logPrefix, name, err)
			failures = append(failures, mcpoauth.FormatConnectFailure(name, err))
			continue
		}
		c.mu.Lock()
		c.sessions[name] = session
		c.mu.Unlock()

		listCtx, listCancel := context.WithTimeout(ctx, defaultConnectTimeout)
		tools, err := session.ListTools(listCtx, &mcp.ListToolsParams{})
		listCancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s list tools %q: %v\n", c.logPrefix, name, err)
			_ = session.Close()
			c.mu.Lock()
			delete(c.sessions, name)
			c.mu.Unlock()
			failures = append(failures, mcpoauth.FormatConnectFailure(name, err))
			continue
		}
		for _, tool := range tools.Tools {
			if tool == nil || strings.TrimSpace(tool.Name) == "" {
				continue
			}
			entry := listedTool{
				server:      name,
				bareName:    tool.Name,
				description: tool.Description,
			}
			if tool.InputSchema != nil {
				var schema map[string]any
				if b, err := json.Marshal(tool.InputSchema); err == nil {
					_ = json.Unmarshal(b, &schema)
				}
				entry.inputSchema = schema
			}
			if uri := toolUIResourceURI(tool); uri != "" {
				entry.uiURI = uri
			}
			listed = append(listed, entry)
		}
	}

	bareCounts := map[string]int{}
	for _, t := range listed {
		bareCounts[t.bareName]++
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, t := range listed {
		collision := bareCounts[t.bareName] > 1
		displayName := QualifiedToolName(t.server, t.bareName, collision)
		entry := toolEntry{
			displayName: displayName,
			bareName:    t.bareName,
			server:      t.server,
			description: t.description,
			inputSchema: t.inputSchema,
		}
		c.tools = append(c.tools, Tool{
			Name:        displayName,
			Description: t.description,
			Server:      t.server,
			InputSchema: t.inputSchema,
		})
		c.toolMap[displayName] = entry
		if !collision {
			c.toolMap[t.bareName] = entry
		}
		if t.uiURI != "" {
			c.toolUI[displayName] = t.uiURI
			c.toolServer[displayName] = t.server
			c.toolUI[t.bareName] = t.uiURI
			c.toolServer[t.bareName] = t.server
		}
	}
	return failures
}

func (c *Client) connectWithTimeout(ctx context.Context, srv model.ADLMCPServer, timeout time.Duration) (*mcp.ClientSession, error) {
	if timeout <= 0 {
		return c.connectOne(srv)
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type connectResult struct {
		session *mcp.ClientSession
		err     error
	}
	ch := make(chan connectResult, 1)
	go func() {
		session, err := c.connectOne(srv)
		ch <- connectResult{session: session, err: err}
	}()

	select {
	case <-waitCtx.Done():
		return nil, waitCtx.Err()
	case res := <-ch:
		return res.session, res.err
	}
}

func (c *Client) connectOne(srv model.ADLMCPServer) (*mcp.ClientSession, error) {
	if cmd := strings.TrimSpace(srv.Command); cmd != "" {
		client := mcp.NewClient(&mcp.Implementation{Name: "nui", Version: "1.0.0"}, nil)
		// Use exec.Command (not CommandContext): connect timeout must not bind to child lifetime.
		command := exec.Command(cmd, srv.Args...)
		if len(srv.Env) > 0 {
			command.Env = envWithOverrides(srv.Env)
		}
		transport := &mcp.CommandTransport{Command: command}
		ctx := context.Background()
		return client.Connect(ctx, transport, nil)
	}
	return mcpoauth.ConnectRemote(context.Background(), srv)
}

// Tools returns a copy of connected tools.
func (c *Client) Tools() []Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Tool, len(c.tools))
	copy(out, c.tools)
	return out
}

// CallTool executes a tool by display or bare name.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	entry, session, ok := c.resolveTool(name)
	if !ok || session == nil {
		return "", fmt.Errorf("tool %q not found", name)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      entry.bareName,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	return FormatCallResult(result), nil
}

// LookupToolUI returns the MCP App resource URI and server name for a tool.
func (c *Client) LookupToolUI(toolName string) (resourceURI, serverName string, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if uri, found := c.toolUI[toolName]; found {
		return uri, c.toolServer[toolName], true
	}
	bare := BareToolName(toolName)
	if uri, found := c.toolUI[bare]; found {
		return uri, c.toolServer[bare], true
	}
	return "", "", false
}

// ReadResource reads an MCP resource from the named server.
func (c *Client) ReadResource(ctx context.Context, serverName, uri string) (string, error) {
	c.mu.RLock()
	session := c.sessions[serverName]
	c.mu.RUnlock()
	if session == nil {
		return "", fmt.Errorf("MCP server %q not found", serverName)
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
	return "", fmt.Errorf("no text content in resource")
}

// CallToolStructured executes a tool and returns a structured result map.
func (c *Client) CallToolStructured(ctx context.Context, serverName, name string, args map[string]any) (map[string]any, error) {
	c.mu.RLock()
	session := c.sessions[serverName]
	c.mu.RUnlock()
	if session == nil {
		return nil, fmt.Errorf("MCP server %q not found", serverName)
	}
	bare := BareToolName(name)
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      bare,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}
	return StructuredCallResult(result), nil
}

func (c *Client) resolveTool(name string) (toolEntry, *mcp.ClientSession, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.toolMap[name]; ok {
		return entry, c.sessions[entry.server], true
	}
	bare := BareToolName(name)
	if entry, ok := c.toolMap[bare]; ok {
		return entry, c.sessions[entry.server], true
	}
	return toolEntry{}, nil, false
}

// Close tears down all MCP server connections.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for name, session := range c.sessions {
		if session != nil {
			_ = session.Close()
		}
		delete(c.sessions, name)
	}
	c.tools = nil
	c.toolMap = map[string]toolEntry{}
	c.toolUI = map[string]string{}
	c.toolServer = map[string]string{}
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

// FormatCallResult stringifies an MCP tool result for LLM tool messages.
func FormatCallResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	if result.StructuredContent != nil {
		b, _ := json.Marshal(result.StructuredContent)
		return string(b)
	}
	var parts []string
	for _, content := range result.Content {
		switch v := content.(type) {
		case *mcp.TextContent:
			parts = append(parts, v.Text)
		default:
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}
	return strings.Join(parts, "\n")
}

// StructuredCallResult converts an MCP tool result to a JSON-friendly map.
func StructuredCallResult(result *mcp.CallToolResult) map[string]any {
	out := map[string]any{
		"isError": result.IsError,
	}
	if result.StructuredContent != nil {
		out["structuredContent"] = result.StructuredContent
	}
	var content []map[string]any
	for _, item := range result.Content {
		switch v := item.(type) {
		case *mcp.TextContent:
			content = append(content, map[string]any{"type": "text", "text": v.Text})
		default:
			content = append(content, map[string]any{"type": "text", "text": fmt.Sprintf("%v", v)})
		}
	}
	out["content"] = content
	return out
}
