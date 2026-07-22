// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nui/internal/model"
)

func TestClientToolNameCollision(t *testing.T) {
	script := filepath.Clean(filepath.Join("..", "..", "dev", "harness-examples", "mock", "e2e_mcp_stdio_server.py"))
	if _, err := os.Stat(script); err != nil {
		t.Skipf("mock mcp server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := New()
	defer client.Close()
	failures := client.ConnectServers(ctx, []model.ADLMCPServer{
		{Name: "alpha", Command: "python3", Args: []string{script}},
		{Name: "beta", Command: "python3", Args: []string{script}},
	})
	if len(failures) != 0 {
		t.Fatalf("ConnectServers: %v", failures)
	}

	tools := client.Tools()
	if len(tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(tools))
	}
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["alpha__ping"] || !names["beta__ping"] {
		t.Fatalf("expected namespaced tools, got %+v", tools)
	}
	if _, err := client.CallTool(ctx, "alpha__ping", nil); err != nil {
		t.Fatalf("qualified call: %v", err)
	}
	if _, err := client.CallTool(ctx, "ping", nil); err == nil {
		t.Fatal("ambiguous bare name should not resolve when namespaced")
	}
}
