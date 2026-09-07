// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"os"
	"testing"
)

func TestRunMCPSubcommandRecognizesBuiltins(t *testing.T) {
	cases := []string{
		"viz-mcp", "agent-mcp", "hitl-mcp", "orchestrator-mcp",
		"skills-mcp", "fs-mcp", "bash-mcp", "mcp",
	}
	for _, name := range cases {
		// Don't actually Run* (blocks on stdio) — only check dispatch recognition.
		if !isMCPSubcommand(name) {
			t.Fatalf("expected %q to be an MCP subcommand", name)
		}
		if !looksLikeMCPSubcommand(name) {
			t.Fatalf("expected %q to look like an MCP subcommand", name)
		}
	}
	if isMCPSubcommand("server") {
		t.Fatal("server should not be treated as MCP subcommand")
	}
	if isMCPSubcommand("") {
		t.Fatal("empty should not be MCP subcommand")
	}
	if !looksLikeMCPSubcommand("future-mcp") {
		t.Fatal("*-mcp should look like MCP even when not yet implemented")
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
