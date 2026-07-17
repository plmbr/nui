// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"

	"nui/internal/model"
	"nui/internal/viz"
)

func loopVizMCPServer() (model.ADLMCPServer, error) {
	exe, err := nuiExecutable()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	return model.ADLMCPServer{
		Name:    viz.MCPName,
		Command: exe,
		Args:    []string{"viz-mcp"},
	}, nil
}

func hasNuiVizMCP(servers []model.ADLMCPServer) bool {
	for _, srv := range servers {
		if strings.TrimSpace(srv.Name) == viz.MCPName {
			return true
		}
	}
	return false
}

func sessionHasnuiVizMCP(configDir string) bool {
	if configDir == "" {
		return false
	}
	return sessionHasNamedMCP(configDir, viz.MCPName)
}

func appendNuiVizMCP(servers []model.ADLMCPServer) ([]model.ADLMCPServer, error) {
	if hasNuiVizMCP(servers) {
		return servers, nil
	}
	srv, err := loopVizMCPServer()
	if err != nil {
		return servers, err
	}
	return append(servers, srv), nil
}

const vizSystemPromptAppendix = `
## Visualizations in nui chat

When the user asks for a chart, graph, table, or dashboard:

1. Do **not** invoke the **Skill** tool or the **dataviz** skill.
2. Build self-contained HTML (Chart.js from a CDN is fine).
3. Call **show_visualization** on the **nui-viz** MCP server with the HTML in the **html** field — in the **same turn**, before any closing text.
4. Never end with "building the chart" or similar without calling **show_visualization** first.
5. After **show_visualization**, do **not** paste markdown images, data:image/... URIs, or base64 in your reply — the chart is already rendered inline in nui chat.

Do **not** use **Write** to save HTML files or tell the user to open a browser tab.
`

func appendVizSystemPrompt(systemPrompt string) string {
	block := strings.TrimSpace(vizSystemPromptAppendix)
	if block == "" {
		return systemPrompt
	}
	base := strings.TrimSpace(systemPrompt)
	if base == "" {
		return block
	}
	return base + "\n\n" + block
}
