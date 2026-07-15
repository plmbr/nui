// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"fmt"
	"strings"

	"loop/internal/store"
)

var loopListenPort = 8080

// SetListenPort records the Loop HTTP listen port for OAuth callback URLs.
func SetListenPort(port int) {
	if port > 0 {
		loopListenPort = port
	}
}

// RedirectURI returns the OAuth callback URL Loop uses.
func RedirectURI() (string, error) {
	settings, err := store.LoadSettings()
	if err == nil {
		if u := strings.TrimSpace(settings.MCPOAuthCallbackURL); u != "" {
			if strings.Contains(u, "/api/mcp-oauth/callback") {
				return strings.TrimRight(u, "/"), nil
			}
			return strings.TrimRight(u, "/") + "/api/mcp-oauth/callback", nil
		}
	}
	return fmt.Sprintf("http://127.0.0.1:%d/api/mcp-oauth/callback", loopListenPort), nil
}
