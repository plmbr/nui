// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/store"
)

func TestSaveDefinitionYAML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	content := `adl: "1.0"
id: test-helper
name: Test Helper
description: A test agent.
harness:
  type: api
  provider: anthropic
systemPrompt: |
  Help the user.
`
	path, err := SaveDefinitionYAML(content, false)
	if err != nil {
		t.Fatal(err)
	}
	agentsDir, err := store.AgentsDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(agentsDir, "test-helper.yaml")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "id: test-helper") {
		t.Fatalf("saved yaml = %q", string(data))
	}

	_, err = SaveDefinitionYAML(content, false)
	if err == nil {
		t.Fatal("expected duplicate error without overwrite")
	}

	path, err = SaveDefinitionYAML(strings.Replace(content, "Help the user.", "Updated.", 1), true)
	if err != nil {
		t.Fatal(err)
	}
	if path != want {
		t.Fatalf("overwrite path = %q", path)
	}
}
