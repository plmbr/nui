// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

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
	"loop/internal/mcpoauth"
	"loop/internal/model"
)

const sessionMCPConnectTimeout = 15 * time.Second

type sessionMCPTool struct {
	Name        string
	Description string
	Server      string
	InputSchema map[string]any
}

// SessionMCP connects to MCP servers declared in session harness deps.
type SessionMCP struct {
	mu       sync.Mutex
	sessions map[string]*mcp.ClientSession
	tools    []sessionMCPTool
	toolMap  map[string]sessionMCPTool
}

// NewSessionMCP creates a session-scoped MCP client (call ConnectServers before use).
func NewSessionMCP() *SessionMCP {
	return &SessionMCP{
		sessions: map[string]*mcp.ClientSession{},
		toolMap:  map[string]sessionMCPTool{},
	}
}

// ConnectServers connects to the given MCP server list.
func (s *SessionMCP) ConnectServers(ctx context.Context, servers []model.ADLMCPServer) error {
	for _, srv := range servers {
		name := strings.TrimSpace(srv.Name)
		if name == "" {
			continue
		}
		// MCP jsonrpc connections are bound to the context passed to Client.Connect.
		// Do not cancel that context until the session is closed.
		session, err := s.connectWithTimeout(ctx, srv, sessionMCPConnectTimeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[api-harness] mcp connect %q: %v\n", name, err)
			continue
		}
		s.mu.Lock()
		s.sessions[name] = session
		s.mu.Unlock()

		listCtx, listCancel := context.WithTimeout(ctx, sessionMCPConnectTimeout)
		tools, err := session.ListTools(listCtx, &mcp.ListToolsParams{})
		listCancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[api-harness] mcp list tools %q: %v\n", name, err)
			_ = session.Close()
			s.mu.Lock()
			delete(s.sessions, name)
			s.mu.Unlock()
			continue
		}
		for _, tool := range tools.Tools {
			if tool == nil || strings.TrimSpace(tool.Name) == "" {
				continue
			}
			entry := sessionMCPTool{
				Name:        tool.Name,
				Description: tool.Description,
				Server:      name,
			}
			if tool.InputSchema != nil {
				var schema map[string]any
				if b, err := json.Marshal(tool.InputSchema); err == nil {
					_ = json.Unmarshal(b, &schema)
				}
				entry.InputSchema = schema
			}
			s.mu.Lock()
			s.tools = append(s.tools, entry)
			s.toolMap[tool.Name] = entry
			s.mu.Unlock()
		}
	}
	return nil
}

func (s *SessionMCP) connectWithTimeout(ctx context.Context, srv model.ADLMCPServer, timeout time.Duration) (*mcp.ClientSession, error) {
	if timeout <= 0 {
		return s.connectOne(ctx, srv)
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type connectResult struct {
		session *mcp.ClientSession
		err     error
	}
	ch := make(chan connectResult, 1)
	go func() {
		session, err := s.connectOne(ctx, srv)
		ch <- connectResult{session: session, err: err}
	}()

	select {
	case <-waitCtx.Done():
		return nil, waitCtx.Err()
	case res := <-ch:
		return res.session, res.err
	}
}

func (s *SessionMCP) connectOne(ctx context.Context, srv model.ADLMCPServer) (*mcp.ClientSession, error) {
	if cmd := strings.TrimSpace(srv.Command); cmd != "" {
		client := mcp.NewClient(&mcp.Implementation{Name: "loop", Version: "1.0.0"}, nil)
		command := exec.CommandContext(ctx, cmd, srv.Args...)
		if len(srv.Env) > 0 {
			command.Env = envWithOverrides(srv.Env)
		}
		transport := &mcp.CommandTransport{Command: command}
		return client.Connect(ctx, transport, nil)
	}
	return mcpoauth.ConnectRemote(ctx, srv)
}

func (s *SessionMCP) Tools() []sessionMCPTool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]sessionMCPTool, len(s.tools))
	copy(out, s.tools)
	return out
}

func (s *SessionMCP) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	s.mu.Lock()
	entry, ok := s.toolMap[name]
	session := s.sessions[entry.Server]
	s.mu.Unlock()
	if !ok || session == nil {
		return "", fmt.Errorf("tool %q not found", name)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	return formatMCPCallResult(result), nil
}

func formatMCPCallResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	if result.StructuredContent != nil {
		b, _ := json.Marshal(result.StructuredContent)
		return string(b)
	}
	var parts []string
	for _, c := range result.Content {
		switch v := c.(type) {
		case *mcp.TextContent:
			parts = append(parts, v.Text)
		default:
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}
	return strings.Join(parts, "\n")
}

func (s *SessionMCP) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, session := range s.sessions {
		if session != nil {
			_ = session.Close()
		}
		delete(s.sessions, name)
	}
}
