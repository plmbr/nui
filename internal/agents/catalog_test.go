// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallAndRemoveLocalAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NUI_HOME", "")

	src := filepath.Join(t.TempDir(), "watchdog.yaml")
	content := []byte(`adl: "1.0"
id: test-watchdog
name: Test Watchdog
harness:
  type: claude-code
`)
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	id, err := Install(src, true)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if id != "test-watchdog" {
		t.Fatalf("id = %q, want test-watchdog", id)
	}

	agentsDir := filepath.Join(home, ".nui", "agents")
	if _, err := os.Stat(filepath.Join(agentsDir, "test-watchdog.yaml")); err != nil {
		t.Fatalf("installed agent file missing: %v", err)
	}

	if err := Remove("test-watchdog"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := Remove("test-watchdog"); err == nil {
		t.Fatal("expected error removing missing agent")
	}
}

func TestRemoveBuiltinAgent(t *testing.T) {
	if err := Remove("claude-code"); err == nil {
		t.Fatal("expected error removing builtin agent")
	}
}
