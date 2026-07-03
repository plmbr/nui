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
	t.Setenv("LOOP_HOME", "")

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

	id, err := Install(src)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if id != "test-watchdog" {
		t.Fatalf("id = %q, want test-watchdog", id)
	}

	types, err := ListTypes()
	if err != nil {
		t.Fatalf("ListTypes: %v", err)
	}
	found := false
	for _, info := range types {
		if info.ID == "test-watchdog" && info.Source == "user" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("installed agent not listed")
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
