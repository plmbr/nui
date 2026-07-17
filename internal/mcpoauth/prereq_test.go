// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nui/internal/model"
)

func TestValidateOAuthStartRequiresClientSecretWhenNoDCR(t *testing.T) {
	srv := newNoDCRTestServer(t)
	defer srv.Close()

	err := validateOAuthStart(context.Background(), model.ADLMCPServer{
		Name: "github",
		URL:  srv.URL + "/mcp",
		Type: "http",
		Auth: &model.ADLMCPServerAuth{ClientID: "my-client"},
	})
	if err == nil {
		t.Fatal("expected error when client secret missing")
	}
	if !strings.Contains(err.Error(), "client secret") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateOAuthStartAllowsPreregisteredClient(t *testing.T) {
	srv := newNoDCRTestServer(t)
	defer srv.Close()

	err := validateOAuthStart(context.Background(), model.ADLMCPServer{
		Name: "github",
		URL:  srv.URL + "/mcp",
		Type: "http",
		Auth: &model.ADLMCPServerAuth{ClientID: "my-client", ClientSecret: "my-secret"},
	})
	if err != nil {
		t.Fatalf("validateOAuthStart = %v", err)
	}
}

func newNoDCRTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	base := srv.URL
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata := fmt.Sprintf(`{
			"issuer": %q,
			"authorization_endpoint": %q,
			"token_endpoint": %q,
			"code_challenge_methods_supported": ["S256"]
		}`, base+"/.well-known/oauth-authorization-server", base+"/authorize", base+"/token")
		prmBody := fmt.Sprintf(`{
			"resource": %q,
			"authorization_servers": [%q],
			"scopes_supported": ["repo"]
		}`, base+"/mcp", base+"/.well-known/oauth-authorization-server")
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer error="invalid_token", resource_metadata=%q`, base+"/.well-known/oauth-protected-resource/mcp"))
			w.WriteHeader(http.StatusUnauthorized)
		case strings.Contains(r.URL.Path, "oauth-protected-resource"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(prmBody))
		case strings.Contains(r.URL.Path, "oauth-authorization-server"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(metadata))
		default:
			http.NotFound(w, r)
		}
	})
	return srv
}

func TestValidateOAuthStartRequiresClientIDWhenNoDCR(t *testing.T) {
	srv := newNoDCRTestServer(t)
	defer srv.Close()

	err := validateOAuthStart(context.Background(), model.ADLMCPServer{
		Name: "github",
		URL:  srv.URL + "/mcp",
		Type: "http",
	})
	if err == nil {
		t.Fatal("expected error when DCR unavailable and no client ID")
	}
	if !strings.Contains(err.Error(), "pre-registered OAuth client") {
		t.Fatalf("error = %v", err)
	}
}
