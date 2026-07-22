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
