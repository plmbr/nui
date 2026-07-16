// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"net/url"
	"strings"

	"loop/internal/model"
)

const (
	loopHitlMCPName  = "loop-hitl"
	loopVizMCPName   = "loop-viz"
	loopAgentMCPName = "loop-agent"
)

// ServerKey returns the canonical key for token storage.
func ServerKey(srv model.ADLMCPServer) string {
	if u := strings.TrimSpace(srv.URL); u != "" {
		return canonicalURL(u)
	}
	return strings.TrimSpace(srv.Name)
}

func canonicalURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}
	u.Fragment = ""
	u.RawFragment = ""
	u.User = nil
	return strings.TrimRight(u.String(), "/")
}

// IsBuiltin reports whether the server is a Loop-injected stdio MCP server.
func IsBuiltin(srv model.ADLMCPServer) bool {
	name := strings.TrimSpace(srv.Name)
	switch name {
	case loopHitlMCPName, loopVizMCPName, loopAgentMCPName:
		return true
	default:
		return strings.HasPrefix(name, "ext-")
	}
}

// IsRemote reports whether the server uses HTTP/SSE transport.
func IsRemote(srv model.ADLMCPServer) bool {
	if strings.TrimSpace(srv.Command) != "" {
		return false
	}
	return strings.TrimSpace(srv.URL) != ""
}

// NeedsOAuthConfig reports whether OAuth may apply to this server.
func NeedsOAuthConfig(srv model.ADLMCPServer) bool {
	if IsBuiltin(srv) || !IsRemote(srv) {
		return false
	}
	if hasStaticAuthHeader(srv) {
		return false
	}
	return srv.Auth != nil
}

func hasStaticAuthHeader(srv model.ADLMCPServer) bool {
	if len(srv.Headers) == 0 {
		return false
	}
	auth, ok := srv.Headers["Authorization"]
	if !ok {
		auth = srv.Headers["authorization"]
	}
	return strings.TrimSpace(auth) != ""
}
