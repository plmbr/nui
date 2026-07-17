// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDevcontainerExecCommand(t *testing.T) {
	cmd := devcontainerExecCommand(context.Background(), "/tmp/nui-session", "claude", []string{"-p", "hi"})
	if filepath.Base(cmd.Path) != "devcontainer" {
		t.Fatalf("path = %q", cmd.Path)
	}
	want := []string{"exec", "--workspace-folder", "/tmp/nui-session", "claude", "-p", "hi"}
	if len(cmd.Args) < len(want)+1 {
		t.Fatalf("args = %v", cmd.Args)
	}
	for i, arg := range want {
		if cmd.Args[i+1] != arg {
			t.Fatalf("args[%d] = %q, want %q", i+1, cmd.Args[i+1], arg)
		}
	}
}
