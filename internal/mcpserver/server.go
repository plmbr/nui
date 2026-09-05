// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"nui/internal/nuiclient"
)

// Run starts the nui MCP server on stdio, proxying to the nui REST API.
func Run(ctx context.Context, baseURL string) error {
	client := nuiclient.New(baseURL)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "nui",
		Version: "1.0.0",
	}, nil)

	registerTools(server, client)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerTools(server *mcp.Server, client *nuiclient.Client) {
	server.AddTool(&mcp.Tool{
		Name:        "list_agents",
		Description: "List available nui agent types (built-in and installed ADL)",
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
		Description: "List nui sessions",
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
		Description: "Create a new nui session",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":         map[string]any{"type": "string"},
				"agent_type":   map[string]any{"type": "string"},
				"working_dir":  map[string]any{"type": "string"},
				"agent_config": map[string]any{"type": "object"},
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
		sess, err := client.CreateSession(ctx, nuiclient.CreateSessionRequest{
			Name:        name,
			AgentType:   agentType,
			WorkingDir:  defaultWorkingDir(stringArg(args, "working_dir")),
			AgentConfig: mapArg(args, "agent_config"),
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
				"agent_type":   map[string]any{"type": "string"},
				"message":      map[string]any{"type": "string"},
				"working_dir":  map[string]any{"type": "string"},
				"session_id":   map[string]any{"type": "string"},
				"name":         map[string]any{"type": "string", "description": "Session name when creating a new session"},
				"agent_config": map[string]any{"type": "object", "description": "Session agentConfig (e.g. userScopeHarnessConfig)"},
				"wait":         map[string]any{"type": "boolean"},
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
			sess, err := client.CreateSession(ctx, nuiclient.CreateSessionRequest{
				Name:        stringArg(args, "name"),
				AgentType:   agentType,
				WorkingDir:  wd,
				AgentConfig: mapArg(args, "agent_config"),
			})
			if err != nil {
				return toolError(err)
			}
			sessionID = sess.ID
		}

		started, err := client.StartRun(ctx, sessionID, nuiclient.StartRunRequest{
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
			out["final"] = compactRunFinal(rec)
		}

		return toolJSON(out)
	})

	server.AddTool(&mcp.Tool{
		Name:        "get_run_events",
		Description: "Stream or poll run events. Supports timeout_seconds, after_seq, max_events, and max_bytes.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id":      map[string]any{"type": "string"},
				"run_id":          map[string]any{"type": "string"},
				"timeout_seconds": map[string]any{"type": "number"},
				"after_seq":       map[string]any{"type": "string", "description": "Last-Event-ID / cursor for resuming"},
				"max_events":      map[string]any{"type": "number"},
				"max_bytes":       map[string]any{"type": "number"},
			},
			"required": []string{"session_id", "run_id"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		sessionID := stringArg(args, "session_id")
		runID := stringArg(args, "run_id")
		afterSeq := stringArg(args, "after_seq")
		maxEvents := intArg(args, "max_events")
		maxBytes := intArg(args, "max_bytes")
		timeoutSec := intArg(args, "timeout_seconds")

		streamCtx := ctx
		cancel := func() {}
		if timeoutSec > 0 {
			streamCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		}
		defer cancel()

		var events []json.RawMessage
		var totalBytes int
		var truncated bool
		var lastSeq string
		rec, err := client.StreamRunEvents(streamCtx, sessionID, runID, afterSeq, func(data []byte) {
			if maxEvents > 0 && len(events) >= maxEvents {
				truncated = true
				return
			}
			if maxBytes > 0 && totalBytes+len(data) > maxBytes {
				truncated = true
				return
			}
			events = append(events, append(json.RawMessage(nil), data...))
			totalBytes += len(data)
			lastSeq = string(data) // best-effort; clients may prefer GetRun for recovery
		})
		if err != nil && streamCtx.Err() == nil {
			return toolError(err)
		}
		if rec.Status == "" {
			if r, gerr := client.GetRun(ctx, sessionID, runID); gerr == nil {
				rec = r
			}
		}
		out := map[string]any{
			"run":            rec,
			"events":         events,
			"final":          compactRunFinal(rec),
			"truncated":     truncated,
			"eventCount":     len(events),
			"byteCount":      totalBytes,
			"timedOut":       streamCtx.Err() != nil,
			"recoveryHint":   "use get_run for compact final output if the event stream overflowed",
		}
		if lastSeq != "" {
			out["lastSeq"] = afterSeq
		}
		return toolJSON(out)
	})

	server.AddTool(&mcp.Tool{
		Name:        "get_run",
		Description: "Get status and compact final output for a run",
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
		return toolJSON(map[string]any{
			"run":   rec,
			"final": compactRunFinal(rec),
		})
	})

	server.AddTool(&mcp.Tool{
		Name:        "get_run_snapshot",
		Description: "Non-blocking snapshot of run status and compact final output",
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
		return toolJSON(map[string]any{
			"status": rec.Status,
			"final":  compactRunFinal(rec),
			"error":  rec.Error,
			"runId":  rec.RunID,
		})
	})

	server.AddTool(&mcp.Tool{
		Name:        "wait_for_runs",
		Description: "Wait for multiple runs to finish; returns compact aggregate status",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"runs": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"session_id": map[string]any{"type": "string"},
							"run_id":     map[string]any{"type": "string"},
						},
						"required": []string{"session_id", "run_id"},
					},
				},
				"timeout_seconds": map[string]any{"type": "number"},
			},
			"required": []string{"runs"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		rawRuns, _ := args["runs"].([]any)
		timeoutSec := intArg(args, "timeout_seconds")
		waitCtx := ctx
		cancel := func() {}
		if timeoutSec > 0 {
			waitCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		}
		defer cancel()

		type runRef struct {
			SessionID string
			RunID     string
		}
		var refs []runRef
		for _, item := range rawRuns {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			refs = append(refs, runRef{
				SessionID: stringArg(m, "session_id"),
				RunID:     stringArg(m, "run_id"),
			})
		}
		if len(refs) == 0 {
			return toolError(fmt.Errorf("runs required"))
		}

		results := make([]map[string]any, len(refs))
		var wg sync.WaitGroup
		for i, ref := range refs {
			wg.Add(1)
			go func(i int, ref runRef) {
				defer wg.Done()
				rec, err := client.WaitRun(waitCtx, ref.SessionID, ref.RunID, 500*time.Millisecond)
				entry := map[string]any{
					"sessionId": ref.SessionID,
					"runId":     ref.RunID,
				}
				if err != nil {
					entry["error"] = err.Error()
					entry["timedOut"] = waitCtx.Err() != nil
					if r, gerr := client.GetRun(ctx, ref.SessionID, ref.RunID); gerr == nil {
						entry["status"] = r.Status
						entry["final"] = compactRunFinal(r)
					}
				} else {
					entry["status"] = rec.Status
					entry["final"] = compactRunFinal(rec)
					entry["error"] = rec.Error
				}
				results[i] = entry
			}(i, ref)
		}
		wg.Wait()

		completed, failed, pending := 0, 0, 0
		for _, r := range results {
			switch strings.TrimSpace(fmt.Sprint(r["status"])) {
			case "completed":
				completed++
			case "failed", "cancelled":
				failed++
			default:
				pending++
			}
		}
		return toolJSON(map[string]any{
			"runs":      results,
			"completed": completed,
			"failed":    failed,
			"pending":   pending,
			"timedOut":  waitCtx.Err() != nil,
		})
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

func compactRunFinal(rec nuiclient.RunRecord) map[string]any {
	out := strings.TrimSpace(rec.Output)
	const max = 32_000
	truncated := false
	if len(out) > max {
		out = out[:max] + "…"
		truncated = true
	}
	return map[string]any{
		"status":    rec.Status,
		"output":    out,
		"error":     rec.Error,
		"truncated": truncated,
	}
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
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func mapArg(args map[string]any, key string) map[string]any {
	v, ok := args[key]
	if !ok || v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func toolJSON(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolError(err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, nil
}

func toolError(err error) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		IsError: true,
	}, nil
}
