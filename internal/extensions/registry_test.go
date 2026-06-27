// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"os"
	"path/filepath"
	"testing"

	"loop/internal/extensions"
)

func TestLoadRegistryFromFiles(t *testing.T) {
	home := t.TempDir()
	extRoot := filepath.Join(home, ".loop", "extensions")
	extDir := filepath.Join(extRoot, "corp-pack")
	if err := os.MkdirAll(filepath.Join(extDir, "skills", "code-review"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "skills", "code-review", "SKILL.md"), []byte("# Code Review\n"), 0644); err != nil {
		t.Fatal(err)
	}

	manifest := `apiVersion: loop.dev/extension/v1
name: corp-pack
version: 1.0.0
displayName: Corp Pack
contributions:
  harnesses:
    source:
      file: harnesses.yaml
    runtime:
      transport: stdio
      command: ["python3", "harness_host.py"]
  catalog:
    mcpServers:
      source:
        file: mcp-servers.json
    skills:
      source:
        file: skills.yaml
  agents:
    source:
      file: agents.yaml
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "harnesses.yaml"), []byte(`harnesses:
  - id: echo
    displayName: Echo
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "mcp-servers.json"), []byte(`{
  "mcpServers": [{"name": "docs", "url": "http://localhost:3040/mcp", "type": "http"}]
}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "skills.yaml"), []byte(`skills:
  - name: code-review
    path: ./skills/code-review
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "agents.yaml"), []byte(`agents:
  - id: reviewer
    name: Reviewer
    harness:
      type: claude-code
`), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)

	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ext, ok := reg.Get("corp-pack")
	if !ok {
		t.Fatal("corp-pack not loaded")
	}
	if len(ext.Harnesses) != 1 || ext.Harnesses[0].ID != "echo" {
		t.Fatalf("harnesses: %+v", ext.Harnesses)
	}
	if len(ext.MCPServers) != 1 || ext.MCPServers[0].Name != "docs" {
		t.Fatalf("mcpServers: %+v", ext.MCPServers)
	}
	if len(ext.Skills) != 1 {
		t.Fatalf("skills: %+v", ext.Skills)
	}
	if len(ext.Agents) != 1 || ext.Agents[0].ID != "ext:corp-pack/reviewer" {
		t.Fatalf("agents: %+v", ext.Agents)
	}

	mcp, err := reg.ResolveMCPRef("ext:corp-pack/docs")
	if err != nil {
		t.Fatal(err)
	}
	if mcp.URL != "http://localhost:3040/mcp" {
		t.Fatalf("mcp url: %q", mcp.URL)
	}

	_, dir, err := reg.ResolveSkill("ext:corp-pack/code-review")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	ref, ok := reg.ResolveHarness("ext:corp-pack/echo")
	if !ok {
		t.Fatal("harness not resolved")
	}
	if ref.Runtime.Transport != "stdio" {
		t.Fatalf("transport: %q", ref.Runtime.Transport)
	}
}

func TestParseExtRef(t *testing.T) {
	ext, item, ok := extensions.ParseExtRef("ext:corp-pack/echo")
	if !ok || ext != "corp-pack" || item != "echo" {
		t.Fatalf("parse: %q %q %v", ext, item, ok)
	}
}

func TestLoadRegistryLegacyCatalogKeys(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".loop", "extensions", "legacy-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: loop.dev/extension/v1
name: legacy-pack
version: 1.0.0
contributions:
  mcpServers:
    source:
      file: mcp-servers.json
  skills:
    source:
      file: skills.yaml
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "mcp-servers.json"), []byte(`{"mcpServers":[{"name":"x","url":"http://localhost/mcp","type":"http"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "skills.yaml"), []byte(`skills:
  - name: s
    content: |
      ---
      name: s
      ---
      hi
`), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ext, ok := reg.Get("legacy-pack")
	if !ok {
		t.Fatal("legacy-pack not loaded")
	}
	if len(ext.MCPServers) != 1 || ext.MCPServers[0].Name != "x" {
		t.Fatalf("mcpServers: %+v", ext.MCPServers)
	}
	if len(ext.Skills) != 1 {
		t.Fatalf("skills: %+v", ext.Skills)
	}
}
