// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpclient

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"nui/internal/model"
)

func TestClientConnectNuiViz(t *testing.T) {
	exe := os.Getenv("NUI_MCP_BINARY")
	if exe == "" {
		tmp := filepath.Join(t.TempDir(), "nui")
		cmd := exec.Command("go", "build", "-o", tmp, ".")
		cmd.Dir = filepath.Clean(filepath.Join("..", ".."))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("build nui binary: %v\n%s", err, out)
		}
		exe = tmp
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := New()
	defer client.Close()
	if failures := client.ConnectServers(ctx, []model.ADLMCPServer{{
		Name:    "nui-viz",
		Command: exe,
		Args:    []string{"viz-mcp"},
	}}); len(failures) != 0 {
		t.Fatalf("ConnectServers: %v", failures)
	}
	tools := client.Tools()
	if len(tools) == 0 {
		t.Fatal("expected tools")
	}
	found := false
	for _, tool := range tools {
		if tool.Name == "show_visualization" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tools = %+v", tools)
	}
}

func TestClientConnectE2EMCP(t *testing.T) {
	script := filepath.Clean(filepath.Join("..", "..", "dev", "harness-examples", "mock", "e2e_mcp_stdio_server.py"))
	if _, err := os.Stat(script); err != nil {
		t.Skipf("mock mcp server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := New()
	defer client.Close()
	failures := client.ConnectServers(ctx, []model.ADLMCPServer{{
		Name:    "e2e",
		Command: "python3",
		Args:    []string{script},
	}})
	if len(failures) != 0 {
		t.Fatalf("ConnectServers: %v", failures)
	}
	result, err := client.CallTool(ctx, "ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "pong" {
		t.Fatalf("result = %q", result)
	}
}
