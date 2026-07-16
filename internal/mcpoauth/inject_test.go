// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"loop/internal/model"
)

func TestServerKeyPrefersURL(t *testing.T) {
	srv := model.ADLMCPServer{Name: "linear", URL: "https://mcp.example.com/mcp/"}
	if got := ServerKey(srv); got != "https://mcp.example.com/mcp" {
		t.Fatalf("ServerKey = %q", got)
	}
}

func TestIsBuiltin(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{loopHitlMCPName, true},
		{loopVizMCPName, true},
		{loopAgentMCPName, true},
		{"ext-corp-tools", true},
		{"my-remote", false},
	}
	for _, tc := range cases {
		if got := IsBuiltin(model.ADLMCPServer{Name: tc.name}); got != tc.want {
			t.Fatalf("IsBuiltin(%q) = %v", tc.name, got)
		}
	}
}

func TestResolveServersSkipsBuiltins(t *testing.T) {
	servers := []model.ADLMCPServer{
		{Name: loopHitlMCPName, Command: "loop", Args: []string{"hitl-mcp"}},
		{Name: "remote", URL: "https://example.com/mcp", Auth: &model.ADLMCPServerAuth{ClientID: "id"}},
	}
	res := ResolveServers(servers)
	if len(res) != 2 {
		t.Fatalf("servers len = %d", len(res))
	}
}

func TestResolveServersInjectsBearerToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	SetTokensPathForTest(path)
	t.Cleanup(func() { SetTokensPathForTest("") })

	key := "https://auth.example.com/mcp"
	if err := SaveToken(key, storedCredential{
		Token: &oauth2.Token{
			AccessToken: "secret-token",
			Expiry:      time.Now().Add(time.Hour),
		},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = DeleteToken(key) })

	servers := []model.ADLMCPServer{{Name: "svc", URL: key}}
	res := ResolveServers(servers)
	if got := res[0].Headers["Authorization"]; got != "Bearer secret-token" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestNeedsAuthStatusCodes(t *testing.T) {
	if NeedsAuth(nil) {
		t.Fatal("nil response should not need auth")
	}
}

func TestNeedsOAuthConfig(t *testing.T) {
	remote := model.ADLMCPServer{Name: "analytics", URL: "https://example.com/mcp"}
	if NeedsOAuthConfig(remote) {
		t.Fatal("remote server without auth config should not need OAuth config")
	}
	withAuth := model.ADLMCPServer{
		Name: "github",
		URL:  "https://example.com/mcp",
		Auth: &model.ADLMCPServerAuth{ClientID: "id"},
	}
	if !NeedsOAuthConfig(withAuth) {
		t.Fatal("server with auth config should need OAuth config")
	}
}

func TestStatusForServer(t *testing.T) {
	if got := StatusForServer(model.ADLMCPServer{Name: loopHitlMCPName, Command: "loop"}); got != AuthStatusNotApplicable {
		t.Fatalf("builtin status = %q", got)
	}
	remote := model.ADLMCPServer{Name: "analytics", URL: "https://example.com/mcp"}
	if got := StatusForServer(remote); got != AuthStatusNotApplicable {
		t.Fatalf("remote without auth status = %q, want not_applicable", got)
	}
	withAuth := model.ADLMCPServer{
		Name: "github",
		URL:  "https://example.com/mcp",
		Auth: &model.ADLMCPServerAuth{ClientID: "id"},
	}
	if got := StatusForServer(withAuth); got != AuthStatusNeedsAuth {
		t.Fatalf("remote with auth config status = %q, want needs_auth", got)
	}
}
