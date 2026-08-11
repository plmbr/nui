// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"nui/internal/mcpserver"
	"nui/internal/nuiclient"
)

// runMCPSubcommand handles harness-spawned stdio MCP modes.
// Harnesses invoke os.Executable() with args like "viz-mcp"; the desktop
// binary must speak MCP on stdio instead of opening another GUI window
// (and must not take the single-instance lock).
func runMCPSubcommand(args []string) (handled bool, err error) {
	if len(args) == 0 || !isMCPSubcommand(args[0]) {
		return false, nil
	}
	cmd := strings.TrimSpace(args[0])
	ctx := context.Background()
	switch cmd {
	case "viz-mcp":
		return true, mcpserver.RunViz(ctx)
	case "agent-mcp":
		return true, mcpserver.RunAgent(ctx)
	case "hitl-mcp":
		return true, mcpserver.RunHITL(ctx, mcpBaseURL(args[1:]))
	case "orchestrator-mcp":
		return true, mcpserver.RunOrchestrator(ctx, mcpBaseURL(args[1:]))
	case "mcp":
		return true, mcpserver.Run(ctx, mcpBaseURL(args[1:]))
	default:
		return false, nil
	}
}

func isMCPSubcommand(name string) bool {
	switch strings.TrimSpace(name) {
	case "viz-mcp", "agent-mcp", "hitl-mcp", "orchestrator-mcp", "mcp":
		return true
	default:
		return false
	}
}

func mcpBaseURL(args []string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--url" {
			if u := strings.TrimSpace(args[i+1]); u != "" {
				return strings.TrimRight(u, "/")
			}
		}
	}
	if u := strings.TrimSpace(os.Getenv("NUI_API_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	if u := strings.TrimSpace(os.Getenv("NUI_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	return nuiclient.New("").BaseURL
}

func mcpMain() {
	handled, err := runMCPSubcommand(os.Args[1:])
	if !handled {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "nui desktop mcp: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
