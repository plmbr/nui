// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package devcontainer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProvisionSession(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(filepath.Join(home, ".nui", "sessions"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	workingDir := filepath.Join(dir, "workspace")
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		t.Fatal(err)
	}
	sessionConfig := filepath.Join(dir, "session-config")
	if err := os.MkdirAll(sessionConfig, 0700); err != nil {
		t.Fatal(err)
	}

	managedDir, err := ProvisionSession(ProvisionOpts{
		SessionID:        "sess-1",
		InnerHarness:     "claude-code",
		WorkingDir:       workingDir,
		SessionConfigDir: sessionConfig,
	})
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(managedDir, ".devcontainer", devcontainerJSONName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var spec devcontainerSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Image != DefaultImages["claude-code"] {
		t.Fatalf("image = %q", spec.Image)
	}
	if spec.WorkspaceFolder != containerWorkspace {
		t.Fatalf("workspaceFolder = %q, want %q", spec.WorkspaceFolder, containerWorkspace)
	}
	wantMount := "source=" + workingDir + ",target=" + containerWorkspace + ",type=bind"
	if spec.Mounts[0] != wantMount {
		t.Fatalf("mounts[0] = %q, want %q", spec.Mounts[0], wantMount)
	}
	if len(spec.Mounts) < 2 {
		t.Fatalf("mounts = %v", spec.Mounts)
	}
}

func TestDefaultImages(t *testing.T) {
	for _, harness := range []string{"claude-code", "pi", "codex", "opencode"} {
		if DefaultImages[harness] == "" {
			t.Fatalf("missing default image for %q", harness)
		}
	}
}
