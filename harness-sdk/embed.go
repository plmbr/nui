// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package harnesssdk

import "embed"

// Author-facing Python modules copied to ~/.nui/harness-sdk/ on first use.
//
//go:embed nui_extension.py nui_agent_stdio.py nui_agent.py nui_catalog.py nui_hitl.py nui_hitl_channel.py nui_mention.py nui_mcp_tools.py nui_storage.py
var embedded embed.FS

// FileNames lists embedded harness-sdk modules (stable order for tests and CLI output).
var FileNames = []string{
	"nui_extension.py",
	"nui_agent_stdio.py",
	"nui_agent.py",
	"nui_catalog.py",
	"nui_hitl.py",
	"nui_hitl_channel.py",
	"nui_mention.py",
	"nui_mcp_tools.py",
	"nui_storage.py",
}
