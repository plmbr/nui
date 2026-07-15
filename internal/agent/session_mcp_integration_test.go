// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"loop/internal/model"
)

func TestSessionMCPConnectLoopViz(t *testing.T) {
	exe := os.Getenv("LOOP_MCP_BINARY")
	if exe == "" {
		// Build loop from module root (parent of internal/agent).
		tmp := filepath.Join(t.TempDir(), "loop")
		cmd := exec.Command("go", "build", "-o", tmp, ".")
		cmd.Dir = filepath.Clean(filepath.Join("..", ".."))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("build loop binary: %v\n%s", err, out)
		}
		exe = tmp
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := NewSessionMCP()
	defer client.Close()
	if err := client.ConnectServers(ctx, []model.ADLMCPServer{{
		Name:    "loop-viz",
		Command: exe,
		Args:    []string{"viz-mcp"},
	}}); err != nil {
		t.Fatalf("ConnectServers: %v", err)
	}
	tools := client.Tools()
	if len(tools) == 0 {
		t.Fatal("expected show_visualization tool")
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
