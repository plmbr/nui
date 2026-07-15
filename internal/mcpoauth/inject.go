// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"fmt"
	"strings"

	"loop/internal/model"
)

// AuthStatus describes MCP OAuth state for a server.
type AuthStatus string

const (
	AuthStatusConnected     AuthStatus = "connected"
	AuthStatusNeedsAuth     AuthStatus = "needs_auth"
	AuthStatusExpired       AuthStatus = "expired"
	AuthStatusNotApplicable AuthStatus = "not_applicable"
)

// ResolveResult holds resolved MCP servers and auth warnings.
type ResolveResult struct {
	Servers  []model.ADLMCPServer
	Warnings []string
}

// ResolveServers injects bearer tokens from the OAuth store into remote MCP server headers.
func ResolveServers(servers []model.ADLMCPServer) ResolveResult {
	out := make([]model.ADLMCPServer, 0, len(servers))
	var warnings []string
	for _, srv := range servers {
		resolved, warn := resolveOne(srv)
		out = append(out, resolved)
		if warn != "" {
			warnings = append(warnings, warn)
		}
	}
	return ResolveResult{Servers: out, Warnings: warnings}
}

func resolveOne(srv model.ADLMCPServer) (model.ADLMCPServer, string) {
	if IsBuiltin(srv) || !IsRemote(srv) {
		return srv, ""
	}
	if hasStaticAuthHeader(srv) {
		return srv, ""
	}
	key := ServerKey(srv)
	if HasValidToken(key) {
		cred, _ := LoadToken(key)
		headers := map[string]string{}
		for k, v := range srv.Headers {
			headers[k] = v
		}
		headers["Authorization"] = "Bearer " + strings.TrimSpace(cred.Token.AccessToken)
		srv.Headers = headers
		return srv, ""
	}
	if srv.Auth != nil || NeedsOAuthConfig(srv) {
		name := strings.TrimSpace(srv.Name)
		if name == "" {
			name = key
		}
		return srv, fmt.Sprintf("MCP server %q needs authentication — connect in Customize → MCP Servers", name)
	}
	return srv, ""
}

// TokenAuthStatus returns OAuth status for a server key using stored tokens only.
func TokenAuthStatus(serverKey string) (AuthStatus, bool) {
	key := strings.TrimSpace(serverKey)
	if key == "" {
		return "", false
	}
	if HasValidToken(key) {
		return AuthStatusConnected, true
	}
	cred, ok := LoadToken(key)
	if ok && cred.Token != nil && strings.TrimSpace(cred.Token.AccessToken) != "" {
		return AuthStatusExpired, true
	}
	return "", false
}

// StatusForServer returns the OAuth status for a server definition.
func StatusForServer(srv model.ADLMCPServer) AuthStatus {
	if IsBuiltin(srv) || !IsRemote(srv) {
		return AuthStatusNotApplicable
	}
	if hasStaticAuthHeader(srv) {
		return AuthStatusConnected
	}
	key := ServerKey(srv)
	if HasValidToken(key) {
		return AuthStatusConnected
	}
	cred, ok := LoadToken(key)
	if ok && cred.Token != nil && strings.TrimSpace(cred.Token.AccessToken) != "" {
		return AuthStatusExpired
	}
	if srv.Auth != nil || NeedsOAuthConfig(srv) {
		return AuthStatusNeedsAuth
	}
	return AuthStatusNotApplicable
}

// BearerToken returns a valid access token for the server, if any.
func BearerToken(srv model.ADLMCPServer) (string, bool) {
	key := ServerKey(srv)
	cred, ok := LoadToken(key)
	if !ok || cred.Token == nil {
		return "", false
	}
	tok := strings.TrimSpace(cred.Token.AccessToken)
	if tok == "" {
		return "", false
	}
	if !cred.Token.Expiry.IsZero() && !cred.Token.Expiry.After(cred.UpdatedAt) && !HasValidToken(key) {
		return "", false
	}
	return tok, HasValidToken(key) || tok != ""
}
