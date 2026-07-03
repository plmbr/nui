// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"loop/internal/loopclient"
)

// Run starts the Loop MCP server on stdio, proxying to the Loop REST API.
func Run(ctx context.Context, baseURL string) error {
	client := loopclient.New(baseURL)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "loop",
		Version: "1.0.0",
	}, nil)

	registerTools(server, client)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerTools(server *mcp.Server, client *loopclient.Client) {
	server.AddTool(&mcp.Tool{
		Name:        "list_agents",
		Description: "List available Loop agent types (builtin and custom ADL)",
		InputSchema: emptyObjectSchema(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agents, err := client.ListAgents(ctx)
		if err != nil {
			return toolError(err)
		}
		return toolJSON(agents)
	})

	server.AddTool(&mcp.Tool{
		Name:        "list_sessions",
		Description: "List Loop sessions",
		InputSchema: emptyObjectSchema(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessions, err := client.ListSessions(ctx)
		if err != nil {
			return toolError(err)
		}
		return toolJSON(sessions)
	})

	server.AddTool(&mcp.Tool{
		Name:        "create_session",
		Description: "Create a new Loop session",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string"},
				"agent_type":  map[string]any{"type": "string"},
				"working_dir": map[string]any{"type": "string"},
			},
			"required": []string{},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		agentType := stringArg(args, "agent_type")
		if agentType == "" {
			var err error
			agentType, err = client.ResolveDefaultAgentType(ctx)
			if err != nil {
				return toolError(err)
			}
		}
		name := stringArg(args, "name")
		sess, err := client.CreateSession(ctx, loopclient.CreateSessionRequest{
			Name:       name,
			AgentType:  agentType,
			WorkingDir: defaultWorkingDir(stringArg(args, "working_dir")),
		})
		if err != nil {
			return toolError(err)
		}
		return toolJSON(sess)
	})

	server.AddTool(&mcp.Tool{
		Name:        "run_agent",
		Description: "Create a session (unless session_id is set), start an async agent run, and optionally wait for completion",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_type":  map[string]any{"type": "string"},
				"message":     map[string]any{"type": "string"},
				"working_dir": map[string]any{"type": "string"},
				"session_id":  map[string]any{"type": "string"},
				"wait":        map[string]any{"type": "boolean"},
			},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		sessionID := stringArg(args, "session_id")
		if sessionID == "" {
			agentType := stringArg(args, "agent_type")
			if agentType == "" {
				var err error
				agentType, err = client.ResolveDefaultAgentType(ctx)
				if err != nil {
					return toolError(err)
				}
			}
			wd := defaultWorkingDir(stringArg(args, "working_dir"))
			if wd == "" {
				return toolError(fmt.Errorf("working directory required"))
			}
			sess, err := client.CreateSession(ctx, loopclient.CreateSessionRequest{
				AgentType:  agentType,
				WorkingDir: wd,
			})
			if err != nil {
				return toolError(err)
			}
			sessionID = sess.ID
		}

		started, err := client.StartRun(ctx, sessionID, loopclient.StartRunRequest{
			Message: stringArg(args, "message"),
		})
		if err != nil {
			return toolError(err)
		}

		wait := true
		if v, ok := args["wait"].(bool); ok {
			wait = v
		}

		out := map[string]any{
			"sessionId": sessionID,
			"runId":     started.RunID,
			"status":    started.Status,
		}

		if wait {
			rec, err := client.WaitRun(ctx, sessionID, started.RunID, 0)
			if err != nil {
				return toolError(err)
			}
			out["status"] = rec.Status
			out["output"] = rec.Output
			out["error"] = rec.Error
		}

		return toolJSON(out)
	})

	server.AddTool(&mcp.Tool{
		Name:        "get_run_events",
		Description: "Stream run events until the run finishes (returns final status)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string"},
				"run_id":     map[string]any{"type": "string"},
			},
			"required": []string{"session_id", "run_id"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		sessionID := stringArg(args, "session_id")
		runID := stringArg(args, "run_id")
		var events []json.RawMessage
		rec, err := client.StreamRunEvents(ctx, sessionID, runID, "", func(data []byte) {
			events = append(events, append(json.RawMessage(nil), data...))
		})
		if err != nil {
			return toolError(err)
		}
		return toolJSON(map[string]any{
			"run":    rec,
			"events": events,
		})
	})

	server.AddTool(&mcp.Tool{
		Name:        "get_run",
		Description: "Get status and output for a run",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string"},
				"run_id":     map[string]any{"type": "string"},
			},
			"required": []string{"session_id", "run_id"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		rec, err := client.GetRun(ctx, stringArg(args, "session_id"), stringArg(args, "run_id"))
		if err != nil {
			return toolError(err)
		}
		return toolJSON(rec)
	})

	server.AddTool(&mcp.Tool{
		Name:        "stop_run",
		Description: "Cancel an in-flight run for a session",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string"},
				"run_id":     map[string]any{"type": "string"},
			},
			"required": []string{"session_id"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		if err := client.StopRun(ctx, stringArg(args, "session_id"), stringArg(args, "run_id")); err != nil {
			return toolError(err)
		}
		return toolJSON(map[string]bool{"ok": true})
	})
}

func emptyObjectSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func defaultWorkingDir(requested string) string {
	if wd := strings.TrimSpace(requested); wd != "" {
		return wd
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func parseArgs(req *mcp.CallToolRequest) map[string]any {
	if req == nil || req.Params == nil || len(req.Params.Arguments) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func toolJSON(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func toolError(err error) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: err.Error()},
		},
		IsError: true,
	}, nil
}
