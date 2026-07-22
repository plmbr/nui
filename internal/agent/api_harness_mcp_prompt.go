// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"fmt"
	"sort"
	"strings"

	"nui/internal/mcpclient"
)

func mcpToolCatalogSystemPrompt(tools []mcpclient.Tool) string {
	if len(tools) == 0 {
		return ""
	}
	byServer := map[string][]string{}
	for _, tool := range tools {
		server := strings.TrimSpace(tool.Server)
		if server == "" {
			server = "mcp"
		}
		byServer[server] = append(byServer[server], tool.Name)
	}
	servers := make([]string, 0, len(byServer))
	for server := range byServer {
		servers = append(servers, server)
	}
	sort.Strings(servers)

	var b strings.Builder
	b.WriteString("## MCP servers and tools (this session)\n\n")
	b.WriteString("When the user asks which MCP servers or tools are available, list the **exact server names and tool names** below. ")
	b.WriteString("Do not invent server or tool names that are not listed below.\n\n")
	for _, server := range servers {
		toolNames := byServer[server]
		sort.Strings(toolNames)
		fmt.Fprintf(&b, "### %s\n", server)
		for _, name := range toolNames {
			fmt.Fprintf(&b, "- `%s`\n", name)
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
