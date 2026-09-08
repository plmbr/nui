// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerUserConfigStagingDir_sessionConfig(t *testing.T) {
	sessionDir := filepath.Join(t.TempDir(), "sessions", "sess-1")
	got, err := dockerUserConfigStagingDir(sessionDir)
	if err != nil {
		t.Fatal(err)
	}
	if got != sessionDir {
		t.Fatalf("got %q want %q", got, sessionDir)
	}
	if st, err := os.Stat(sessionDir); err != nil || !st.IsDir() {
		t.Fatalf("session dir not created: %v", err)
	}
}

func TestDockerUserConfigStagingDir_tempFallback(t *testing.T) {
	got, err := dockerUserConfigStagingDir("")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(got) })
	if !strings.Contains(filepath.Base(got), "nui-docker-cfg-") {
		t.Fatalf("expected nui-docker-cfg temp dir, got %q", got)
	}
	a, err := dockerUserConfigStagingDir("")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(a) })
	if a == got {
		t.Fatal("expected unique temp staging dirs for concurrent sessions")
	}
}

func TestSnapshotJSONFile_writesUnderStagingDir(t *testing.T) {
	home := t.TempDir()
	src := filepath.Join(home, ".claude.json")
	if err := os.WriteFile(src, []byte(`{"mcpServers":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(home, "sessions", "sess-a")
	path := snapshotJSONFile(src, staging, ".claude-snapshot.json")
	if path == "" {
		t.Fatal("expected snapshot path")
	}
	want := filepath.Join(staging, ".claude-snapshot.json")
	if path != want {
		t.Fatalf("path = %q want %q", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"mcpServers":{}}` {
		t.Fatalf("snapshot contents = %q", data)
	}
}

func TestSnapshotJSONFile_rejectsInvalidJSON(t *testing.T) {
	home := t.TempDir()
	src := filepath.Join(home, ".claude.json")
	if err := os.WriteFile(src, []byte(`{not-json`), 0o644); err != nil {
		t.Fatal(err)
	}
	if path := snapshotJSONFile(src, home, ".claude-snapshot.json"); path != "" {
		t.Fatalf("expected empty path for invalid JSON, got %q", path)
	}
}
