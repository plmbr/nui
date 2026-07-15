// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"loop/internal/agents"
)

const loopAgentMCPName = "loop-agent"

// RunAgent starts the loop-agent MCP server on stdio.
func RunAgent(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    loopAgentMCPName,
		Version: "1.0.0",
	}, nil)

	registerAgentTools(server)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerAgentTools(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        "save_agent",
		Description: "Save a Loop ADL agent definition YAML to ~/.loop/agents/ so it appears under Installed agents",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "Complete ADL 1.0 agent YAML",
				},
				"overwrite": map[string]any{
					"type":        "boolean",
					"description": "Replace an existing agent file with the same id when true",
				},
			},
			"required": []string{"content"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		content := stringArg(args, "content")
		overwrite := false
		if v, ok := args["overwrite"].(bool); ok {
			overwrite = v
		}
		path, err := agents.SaveDefinitionYAML(content, overwrite)
		if err != nil {
			return toolError(err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Saved agent to %s", path)},
			},
			StructuredContent: map[string]any{
				"path": path,
			},
		}, nil
	})
}
