// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"nui/internal/model"
)

func TestSessionMCPConnectnuiViz(t *testing.T) {
	exe := os.Getenv("NUI_MCP_BINARY")
	if exe == "" {
		// Build nui from module root (parent of internal/agent).
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

	client := NewSessionMCP()
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
