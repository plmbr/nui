// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"

	"nui/internal/mcpclient"
)

func TestMCPToolCatalogSystemPrompt(t *testing.T) {
	prompt := mcpToolCatalogSystemPrompt([]mcpclient.Tool{
		{Name: "alpha-mcp__ping", Server: "alpha-mcp"},
		{Name: "beta-mcp__echo", Server: "beta-mcp"},
	})
	if !strings.Contains(prompt, "### alpha-mcp") {
		t.Fatalf("missing server section: %q", prompt)
	}
	if !strings.Contains(prompt, "`alpha-mcp__ping`") {
		t.Fatalf("missing tool name: %q", prompt)
	}
	if !strings.Contains(prompt, "Do not invent server or tool names") {
		t.Fatal("should warn against inventing server names")
	}
}

func TestMCPToolCatalogSystemPrompt_orchestratorRouting(t *testing.T) {
	prompt := mcpToolCatalogSystemPrompt([]mcpclient.Tool{
		{Name: "nui-orchestrator__list_agents", Server: "nui-orchestrator"},
		{Name: "nui-orchestrator__launch_session", Server: "nui-orchestrator"},
		{Name: "nui-agent__save_agent", Server: "nui-agent"},
	})
	if !strings.Contains(prompt, "nui-orchestrator__list_agents") {
		t.Fatalf("missing list_agents tool: %q", prompt)
	}
	if !strings.Contains(prompt, "nui-orchestrator__launch_session") {
		t.Fatalf("missing launch_session tool: %q", prompt)
	}
	if !strings.Contains(prompt, "Do not guess agent ids") {
		t.Fatalf("missing routing instructions: %q", prompt)
	}
}
