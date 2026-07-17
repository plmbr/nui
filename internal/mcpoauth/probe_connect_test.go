// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nui/internal/model"
)

func TestProbeConnectFailuresSkipsStdio(t *testing.T) {
	failures := ProbeConnectFailures(context.Background(), []model.ADLMCPServer{
		{Name: "local", Command: "echo", Args: []string{"hi"}},
	})
	if len(failures) != 0 {
		t.Fatalf("failures = %v", failures)
	}
}

func TestProbeConnectFailuresReportsUnreachable(t *testing.T) {
	failures := ProbeConnectFailures(context.Background(), []model.ADLMCPServer{
		{Name: "down", URL: "http://127.0.0.1:1/mcp"},
	})
	if len(failures) != 1 {
		t.Fatalf("failures = %v", failures)
	}
	if !strings.Contains(failures[0], `MCP server "down" failed to connect`) {
		t.Fatalf("failure = %q", failures[0])
	}
}

func TestProbeConnectFailuresReportsUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="mcp"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	failures := ProbeConnectFailures(context.Background(), []model.ADLMCPServer{
		{Name: "secure", URL: srv.URL, Auth: &model.ADLMCPServerAuth{ClientID: "id"}},
	})
	if len(failures) != 1 {
		t.Fatalf("failures = %v", failures)
	}
	if !strings.Contains(failures[0], "needs authentication") {
		t.Fatalf("failure = %q", failures[0])
	}
}

func TestFormatConnectFailureNeedsAuth(t *testing.T) {
	msg := FormatConnectFailure("linear", ErrNeedsAuth)
	if !strings.Contains(msg, "needs authentication") {
		t.Fatalf("msg = %q", msg)
	}
}
