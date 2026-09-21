// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"nui/internal/model"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// TestConnectRemoteIgnoresHangingStandaloneGET covers gateways that accept the
// optional GET SSE after initialize and then never handle the follow-up POST
// for notifications/initialized.
func TestConnectRemoteIgnoresHangingStandaloneGET(t *testing.T) {
	var gotGET atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gotGET.Store(true)
			// Hold like a persistent SSE stream so a later POST would miss the
			// 15s probe deadline if the client opened this GET.
			select {
			case <-r.Context().Done():
			case <-time.After(30 * time.Second):
			}
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if gotGET.Load() {
			select {
			case <-r.Context().Done():
			case <-time.After(30 * time.Second):
			}
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var msg struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(body, &msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch msg.Method {
		case "initialize":
			var params struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			_ = json.Unmarshal(msg.Params, &params)
			version := params.ProtocolVersion
			if version == "" {
				version = "2024-11-05"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(msg.ID),
				"result": map[string]any{
					"protocolVersion": version,
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "mock", "version": "1.0.0"},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(msg.ID),
				"result":  map[string]any{"tools": []any{}},
			})
		default:
			http.Error(w, "unknown method "+msg.Method, http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	session, err := ConnectRemote(ctx, model.ADLMCPServer{Name: "test-mcp", URL: srv.URL})
	if err != nil {
		t.Fatalf("ConnectRemote: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if _, err := session.ListTools(ctx, &mcp.ListToolsParams{}); err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if gotGET.Load() {
		t.Fatal("client opened standalone GET SSE; handshake would hang on this gateway")
	}
}
