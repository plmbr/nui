// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpclient

import "strings"

const toolNameSep = "__"

// QualifiedToolName returns the LLM-visible name for a tool on a server.
func QualifiedToolName(server, bareName string, collision bool) string {
	if collision {
		return server + toolNameSep + bareName
	}
	return bareName
}

// BareToolName strips a server prefix from a qualified tool name.
func BareToolName(name string) string {
	if idx := strings.LastIndex(name, toolNameSep); idx >= 0 {
		return name[idx+len(toolNameSep):]
	}
	return name
}
