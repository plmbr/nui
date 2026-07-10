// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"os"
	"path/filepath"
	"testing"

	"loop/internal/store"
)

func TestLookupDefinitionBuiltin(t *testing.T) {
	def, ok := LookupDefinition("claude-code")
	if !ok {
		t.Fatal("expected builtin claude-code")
	}
	if def.ID != "claude-code" {
		t.Fatalf("got id %q", def.ID)
	}
}

func TestLookupDefinitionUserAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir, err := store.AgentsDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `adl: "1.0"
id: test-eval-agent
name: Test Eval Agent
harness:
  type: claude-code
evals:
  - name: smoke
    input: hi
    expect:
      type: contains
      value: hello
`
	path := filepath.Join(dir, "test-eval-agent.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	def, ok := LookupDefinition("test-eval-agent")
	if !ok {
		t.Fatal("expected user agent")
	}
	if len(def.Evals) != 1 {
		t.Fatalf("got %d evals", len(def.Evals))
	}
	if def.Evals[0].Name != "smoke" {
		t.Fatalf("got eval name %q", def.Evals[0].Name)
	}
}

func TestLookupDefinitionLegacyName(t *testing.T) {
	def, ok := LookupDefinition("Claude Code")
	if !ok || def.ID != "claude-code" {
		t.Fatalf("legacy lookup failed: ok=%v id=%q", ok, def.ID)
	}
}
