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
	case "skills-mcp":
		return true, mcpserver.RunSkills(ctx)
	case "fs-mcp":
		return true, mcpserver.RunFS(ctx)
	case "bash-mcp":
		return true, mcpserver.RunBash(ctx)
	case "mcp":
		return true, mcpserver.Run(ctx, mcpBaseURL(args[1:]))
	default:
		return false, nil
	}
}

func isMCPSubcommand(name string) bool {
	switch strings.TrimSpace(name) {
	case "viz-mcp", "agent-mcp", "hitl-mcp", "orchestrator-mcp",
		"skills-mcp", "fs-mcp", "bash-mcp", "mcp":
		return true
	default:
		return false
	}
}

// looksLikeMCPSubcommand reports argv that should be treated as MCP dispatch
// (never fall through to the GUI, which would print "Listening on…" on stdout).
func looksLikeMCPSubcommand(name string) bool {
	name = strings.TrimSpace(name)
	return name == "mcp" || strings.HasSuffix(name, "-mcp")
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
	if len(os.Args) < 2 {
		return
	}
	arg := os.Args[1]
	handled, err := runMCPSubcommand(os.Args[1:])
	if !handled {
		// Unknown *-mcp / mcp must not open the GUI: server startup prints
		// "Listening on …" to stdout and breaks the MCP JSON-RPC handshake.
		if looksLikeMCPSubcommand(arg) {
			fmt.Fprintf(os.Stderr, "nui desktop mcp: unknown MCP subcommand %q\n", arg)
			os.Exit(1)
		}
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "nui desktop mcp: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
