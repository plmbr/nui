// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nui/internal/extensions"
)

func TestStorageHandlerValidation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "sessionHistory requires agentTypes",
			yaml: `storageHandlers:
  - id: s1
    kind: sessionHistory
`,
			wantErr: "requires agentTypes",
		},
		{
			name: "userMemory forbids agentTypes",
			yaml: `storageHandlers:
  - id: u1
    kind: userMemory
    agentTypes: ["demo"]
`,
			wantErr: "must not set agentTypes",
		},
		{
			name: "invalid kind",
			yaml: `storageHandlers:
  - id: x1
    kind: blobs
`,
			wantErr: "kind must be sessionHistory",
		},
		{
			name: "valid handlers",
			yaml: `storageHandlers:
  - id: s1
    kind: sessionHistory
    agentTypes: ["claude-code"]
  - id: a1
    kind: agentMemory
    agentTypes: ["ext:demo/agent"]
  - id: u1
    kind: userMemory
`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "storage-handlers.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := extensions.LoadStorageHandlersFromFile(path)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("LoadStorageHandlersFromFile() err = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestStorageHandlerRouting(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".nui", "extensions", "storage-pack")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.plmbr.dev/extension/v1
name: storage-pack
version: 1.0.0
contributions:
  storage:
    source:
      file: storage-handlers.yaml
    runtime:
      transport: stdio
      command: ["python3", "storage_host.py"]
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	handlers := `storageHandlers:
  - id: sess
    kind: sessionHistory
    agentTypes: ["claude-code", "ext:storage-pack/agent"]
  - id: agent-mem
    kind: agentMemory
    agentTypes: ["ext:storage-pack/agent"]
  - id: user-mem
    kind: userMemory
`
	if err := os.WriteFile(filepath.Join(extDir, "storage-handlers.yaml"), []byte(handlers), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "storage_host.py"), []byte("#!/usr/bin/env python3\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if !reg.HasSessionHandler("claude-code") {
		t.Fatal("expected session handler for claude-code")
	}
	if reg.HasSessionHandler("pi") {
		t.Fatal("did not expect session handler for pi")
	}
	if !reg.HasAgentMemoryHandler("ext:storage-pack/agent") {
		t.Fatal("expected agent memory handler")
	}
	if !reg.HasUserMemoryHandler() {
		t.Fatal("expected user memory handler")
	}
	if len(reg.SessionHandlers("claude-code")) != 1 {
		t.Fatalf("session handlers: %+v", reg.SessionHandlers("claude-code"))
	}
}
