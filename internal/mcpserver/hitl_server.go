// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"loop/internal/hitl"
	"loop/internal/loopclient"
)

// RunHITL starts the loop-hitl MCP server on stdio.
func RunHITL(ctx context.Context, baseURL string) error {
	client := loopclient.New(baseURL)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "loop-hitl",
		Version: "1.0.0",
	}, nil)

	registerHITLTools(server, client)

	transport := &mcp.StdioTransport{}
	return server.Run(ctx, transport)
}

func registerHITLTools(server *mcp.Server, client *loopclient.Client) {
	server.AddTool(&mcp.Tool{
		Name:        "ask_user",
		Description: "Ask the human a structured question and wait for an answer via Loop UI",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"questions": map[string]any{
					"type":        "array",
					"description": "AskUserQuestion-style question objects",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"question": map[string]any{"type": "string"},
							"header":   map[string]any{"type": "string"},
							"options": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"label":       map[string]any{"type": "string"},
										"description": map[string]any{"type": "string"},
									},
									"required": []string{"label"},
								},
							},
						},
					},
				},
				"title":   map[string]any{"type": "string"},
				"message": map[string]any{"type": "string"},
			},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		payload := map[string]any{
			"questions": args["questions"],
			"title":     args["title"],
			"message":   args["message"],
		}
		return createAndWait(ctx, client, hitl.KindQuestion, payload)
	})

	server.AddTool(&mcp.Tool{
		Name:        "request_approval",
		Description: "Request human approval before continuing (approve/reject)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":       map[string]any{"type": "string"},
				"message":     map[string]any{"type": "string"},
				"toolName":    map[string]any{"type": "string"},
				"toolInput":   map[string]any{"type": "object"},
				"description": map[string]any{"type": "string"},
			},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		payload := map[string]any{
			"title":       args["title"],
			"message":     args["message"],
			"toolName":    args["toolName"],
			"toolInput":   args["toolInput"],
			"description": args["description"],
		}
		return createAndWait(ctx, client, hitl.KindApproval, payload)
	})
}

func createAndWait(ctx context.Context, client *loopclient.Client, kind string, payload map[string]any) (*mcp.CallToolResult, error) {
	sessionID := strings.TrimSpace(os.Getenv("LOOP_SESSION_ID"))
	runID := strings.TrimSpace(os.Getenv("LOOP_RUN_ID"))
	in := hitl.CreateInput{
		SessionID: sessionID,
		RunID:     runID,
		Kind:      kind,
		Payload:   payload,
	}
	req, err := client.CreateHITLRequest(ctx, in)
	if err != nil {
		return toolError(err)
	}
	resp, err := client.WaitHITLRequest(ctx, req.RequestID)
	if err != nil {
		return toolError(err)
	}
	if resp.Status == hitl.StatusDeclined || resp.Status == hitl.StatusCancelled {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("request %s", resp.Status)}},
		}, nil
	}
	return toolJSON(resp)
}
