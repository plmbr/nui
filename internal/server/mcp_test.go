// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapMCPLoad_noConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var m MCPManager
	if err := bootstrapMCPLoad(&m); err != nil {
		t.Fatalf("bootstrapMCPLoad: %v", err)
	}
	if m.clientOrNil() != nil {
		t.Fatalf("client should be nil when no config")
	}
}

func TestMigrateLegacyMCPConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	nuiDir := filepath.Join(home, ".nui")
	if err := os.MkdirAll(nuiDir, 0o700); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(nuiDir, legacyMCPConfigFile)
	newPath := filepath.Join(nuiDir, mcpUIConfigFile)
	body := []byte(`{"mcpServers":{"demo":{"command":"true"}}}`)
	if err := os.WriteFile(oldPath, body, 0o644); err != nil {
		t.Fatal(err)
	}

	migrateLegacyMCPConfig(newPath)

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("legacy .mcp.json should be gone after migration")
	}
	got, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("mcp-ui.json = %q want %q", got, body)
	}

	// Second call is a no-op when mcp-ui.json already exists.
	if err := os.WriteFile(oldPath, []byte(`{"mcpServers":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	migrateLegacyMCPConfig(newPath)
	got, err = os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("existing mcp-ui.json should be preserved, got %q", got)
	}
}

func TestMCPManager_ensureLoaded_idempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var m MCPManager
	if err := m.ensureLoaded(); err != nil {
		t.Fatalf("first ensureLoaded: %v", err)
	}
	if err := m.ensureLoaded(); err != nil {
		t.Fatalf("second ensureLoaded: %v", err)
	}
}
