// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"loop/internal/viz"
)

// RunViz starts the loop-viz MCP server on stdio.
func RunViz(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    viz.MCPName,
		Version: "1.0.0",
	}, nil)

	registerVizTools(server)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerVizTools(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        viz.ToolName,
		Description: "Render an HTML visualization (charts, tables, dashboards) in the Loop chat UI. Provide a complete HTML document or fragment; external libraries (Chart.js, D3, etc.) may be loaded from CDNs.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"html": map[string]any{
					"type":        "string",
					"description": "Complete HTML document or fragment with inline or CDN-linked JavaScript/CSS",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Optional short label shown above the visualization",
				},
			},
			"required": []string{"html"},
		},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		html, title, ok := viz.ParseInput(args)
		if !ok {
			return toolError(fmt.Errorf("html is required"))
		}
		html = viz.PrepareHTML(html)
		msg := "Visualization rendered in Loop UI"
		if title != "" {
			msg = fmt.Sprintf("Visualization rendered: %s", title)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: msg},
			},
			StructuredContent: map[string]any{
				"html":  html,
				"title": title,
			},
		}, nil
	})
}
