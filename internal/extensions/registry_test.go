// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"os"
	"path/filepath"
	"testing"

	"nui/internal/extensions"
)

func TestLoadRegistryFromFiles(t *testing.T) {
	home := t.TempDir()
	extRoot := filepath.Join(home, ".nui", "extensions")
	extDir := filepath.Join(extRoot, "corp-pack")
	if err := os.MkdirAll(filepath.Join(extDir, "skills", "code-review"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "skills", "code-review", "SKILL.md"), []byte("# Code Review\n"), 0644); err != nil {
		t.Fatal(err)
	}

	manifest := `apiVersion: nui.plmbr.dev/extension/v1
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

	infos := reg.Info()
	var corpInfo *extensions.ExtensionInfo
	for i := range infos {
		if infos[i].Name == "corp-pack" {
			corpInfo = &infos[i]
			break
		}
	}
	if corpInfo == nil {
		t.Fatal("corp-pack info missing")
	}
	if len(corpInfo.MCPServerConfigs) != 1 || corpInfo.MCPServerConfigs[0].Name != "docs" {
		t.Fatalf("mcpServerConfigs: %+v", corpInfo.MCPServerConfigs)
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
	extDir := filepath.Join(home, ".nui", "extensions", "legacy-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.plmbr.dev/extension/v1
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

func TestList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	extDir := filepath.Join(home, ".nui", "extensions", "test-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: nui.plmbr.dev/extension/v1
name: test-pack
version: 1.0.0
displayName: Test Pack
description: A test extension
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := extensions.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].Name != "test-pack" || entries[0].DisplayName != "Test Pack" {
		t.Fatalf("entry = %+v", entries[0])
	}
}

func TestLoadRegistryExtraConfigDirUserWins(t *testing.T) {
	home := t.TempDir()
	extra := filepath.Join(t.TempDir(), "extra-config")
	extraExt := filepath.Join(extra, "extensions", "shared-pack")
	userExt := filepath.Join(home, ".nui", "extensions", "shared-pack")
	for _, spec := range []struct {
		dir     string
		version string
	}{
		{extraExt, "1.0.0"},
		{userExt, "2.0.0"},
	} {
		if err := os.MkdirAll(spec.dir, 0755); err != nil {
			t.Fatal(err)
		}
		manifest := `apiVersion: nui.plmbr.dev/extension/v1
name: shared-pack
version: ` + spec.version + `
displayName: Pack
`
		if err := os.WriteFile(filepath.Join(spec.dir, "extension.yaml"), []byte(manifest), 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("HOME", home)
	t.Setenv("NUI_EXTRA_CONFIG_DIRS", extra)

	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ext, ok := reg.Get("shared-pack")
	if !ok {
		t.Fatal("shared-pack not loaded")
	}
	if ext.Manifest.Version != "2.0.0" {
		t.Fatalf("version = %q want user override", ext.Manifest.Version)
	}
	if !ext.Writable {
		t.Fatal("expected writable user extension")
	}
}
