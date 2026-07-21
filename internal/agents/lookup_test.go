// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"os"
	"path/filepath"
	"testing"

	"nui/internal/extensions"
	"nui/internal/store"
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

func TestLookupDefinitionResolvesExtensionAgentHarnessRef(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	extDir := filepath.Join(home, ".nui", "extensions", "corp-pack")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.plmbr.dev/extension/v1
name: corp-pack
version: 1.0.0
contributions:
  agents:
    source:
      file: agents.yaml
`
	agents := `agents:
  - id: mgw-chat-model
    name: Chat Model
    harness:
      type: api
      provider: openai
      model: example/chat-model
      baseUrl: http://example.test/v1
      disableTools: true
`
	for name, content := range map[string]string{
		"extension.yaml": manifest,
		"agents.yaml":    agents,
	} {
		if err := os.WriteFile(filepath.Join(extDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	prev := extensions.Default
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	extensions.Default = reg
	t.Cleanup(func() { extensions.Default = prev })

	agentsDir, err := store.AgentsDir()
	if err != nil {
		t.Fatal(err)
	}
	userAgent := `adl: "1.0"
id: wrapped-agent
name: Wrapped Agent
harness:
  type: ext:corp-pack/mgw-chat-model
`
	if err := os.WriteFile(filepath.Join(agentsDir, "wrapped-agent.yaml"), []byte(userAgent), 0o644); err != nil {
		t.Fatal(err)
	}

	def, ok := LookupDefinition("wrapped-agent")
	if !ok {
		t.Fatal("expected user agent")
	}
	if def.Harness.Type != "api" {
		t.Fatalf("harness.type = %q, want api", def.Harness.Type)
	}
	if def.Harness.Model != "example/chat-model" {
		t.Fatalf("harness.model = %q", def.Harness.Model)
	}
	if !def.Harness.DisableTools {
		t.Fatal("expected disableTools from referenced extension agent")
	}
}
