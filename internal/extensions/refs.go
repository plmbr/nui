// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"strings"
)

const extPrefix = "ext:"

// ParseExtRef splits ext:<extension>/<item> into extension name and item id.
func ParseExtRef(ref string) (extension, item string, ok bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, extPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(ref, extPrefix)
	before, after, found := strings.Cut(rest, "/")
	if !found || before == "" || after == "" {
		return "", "", false
	}
	return before, after, true
}

// IsExtRef reports whether s is an extension reference (ext:...).
func IsExtRef(s string) bool {
	_, _, ok := ParseExtRef(s)
	return ok
}

// HarnessAgentID returns the global agent type id for an extension harness.
func HarnessAgentID(extensionName, harnessID string) string {
	return fmt.Sprintf("%s%s/%s", extPrefix, extensionName, harnessID)
}

// AgentAgentID returns the global agent type id for an extension ADL agent.
func AgentAgentID(extensionName, agentID string) string {
	return fmt.Sprintf("%s%s/%s", extPrefix, extensionName, agentID)
}

// MCPRef returns an ADL MCP ref for an extension server.
func MCPRef(extensionName, serverName string) string {
	return HarnessAgentID(extensionName, serverName)
}

// SkillRef returns an ADL skill ref for an extension skill.
func SkillRef(extensionName, skillName string) string {
	return HarnessAgentID(extensionName, skillName)
}
