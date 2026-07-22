// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"nui/internal/agents"
	"nui/internal/nuiclient"
)

// RunOrchestrator starts the nui-orchestrator MCP server on stdio.
func RunOrchestrator(ctx context.Context, baseURL string) error {
	client := nuiclient.New(baseURL)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "nui-orchestrator",
		Version: "1.0.0",
	}, nil)

	registerOrchestratorTools(server, client)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerOrchestratorTools(server *mcp.Server, client *nuiclient.Client) {
	server.AddTool(&mcp.Tool{
		Name:        "list_agents",
		Description: "List discoverable nui agent types available for new sessions",
		InputSchema: emptyObjectSchema(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		all, err := client.ListAgents(ctx)
		if err != nil {
			return toolError(err)
		}
		var out []map[string]any
		for _, a := range all {
			if !a.Available || a.PromptMode == "auto" {
				continue
			}
			if !agents.IsOrchestratorRoutingTarget(a.ID) {
				continue
			}
			out = append(out, map[string]any{
				"id":          a.ID,
				"label":       a.Label,
				"description": a.Description,
				"harness":     a.Harness,
				"tags":        a.Tags,
			})
		}
		return toolJSON(out)
	})

	server.AddTool(&mcp.Tool{
		Name:        "launch_session",
		Description: "Create a new nui session with the chosen agent and user prompt",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_type":  map[string]any{"type": "string", "description": "Agent id from list_agents"},
				"prompt":      map[string]any{"type": "string", "description": "User prompt to run in the new session"},
				"working_dir": map[string]any{"type": "string", "description": "Working directory (optional)"},
				"name":        map[string]any{"type": "string", "description": "Optional session name"},
			},
			"required": []string{"agent_type", "prompt"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		agentType := stringArg(args, "agent_type")
		prompt := stringArg(args, "prompt")
		if agentType == "" {
			return toolError(fmt.Errorf("agent_type is required"))
		}
		if prompt == "" {
			return toolError(fmt.Errorf("prompt is required"))
		}
		if !agents.IsOrchestratorRoutingTarget(agentType) {
			return toolError(fmt.Errorf("agent %q cannot be used as an orchestrator routing target", agentType))
		}
		available, err := client.ListAgents(ctx)
		if err != nil {
			return toolError(err)
		}
		var found *nuiclient.AgentType
		for i := range available {
			a := available[i]
			if a.ID != agentType {
				continue
			}
			if !a.Available || a.PromptMode == "auto" {
				return toolError(fmt.Errorf("agent %q is not available for launcher sessions", agentType))
			}
			found = &a
			break
		}
		if found == nil {
			return toolError(fmt.Errorf("unknown agent id %q", agentType))
		}
		workingDir := defaultWorkingDir(stringArg(args, "working_dir"))
		if workingDir == "" {
			return toolError(fmt.Errorf("working directory required"))
		}
		sess, err := client.CreateSession(ctx, nuiclient.CreateSessionRequest{
			Name:       stringArg(args, "name"),
			AgentType:  agentType,
			WorkingDir: workingDir,
		})
		if err != nil {
			return toolError(err)
		}
		return toolJSON(map[string]any{
			"session": sess,
			"prompt":  prompt,
		})
	})
}
