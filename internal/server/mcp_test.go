// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
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
