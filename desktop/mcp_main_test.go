// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"os"
	"testing"
)

func TestRunMCPSubcommandRecognizesBuiltins(t *testing.T) {
	cases := []string{"viz-mcp", "agent-mcp", "hitl-mcp", "orchestrator-mcp", "mcp"}
	for _, name := range cases {
		// Don't actually Run* (blocks on stdio) — only check dispatch recognition
		// via a dry parse of the switch by ensuring unknown is false and known
		// would be handled. We call with empty context cancel immediately...
		// Instead verify the command name is in the handled set.
		if !isMCPSubcommand(name) {
			t.Fatalf("expected %q to be an MCP subcommand", name)
		}
	}
	if isMCPSubcommand("server") {
		t.Fatal("server should not be treated as MCP subcommand")
	}
	if isMCPSubcommand("") {
		t.Fatal("empty should not be MCP subcommand")
	}
}

func TestMcpBaseURL(t *testing.T) {
	t.Setenv("NUI_API_URL", "")
	t.Setenv("NUI_URL", "")
	if got := mcpBaseURL([]string{"--url", "http://127.0.0.1:9999"}); got != "http://127.0.0.1:9999" {
		t.Fatalf("flag url: got %q", got)
	}
	t.Setenv("NUI_API_URL", "http://127.0.0.1:7777/")
	if got := mcpBaseURL(nil); got != "http://127.0.0.1:7777" {
		t.Fatalf("env url: got %q", got)
	}
}

func TestMcpMainUnknownFallsThrough(t *testing.T) {
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"nui-desktop"}
	// mcpMain returns without exiting when no subcommand.
	mcpMain()
}
