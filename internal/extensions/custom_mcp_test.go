// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"loop/internal/extensions"
	"loop/internal/model"
)

func TestCustomMCPServerValidation(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".loop", "extensions", "tool-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: loop.dev/extension/v1
name: tool-pack
version: 1.0.0
contributions:
  aiAssets:
    mcpServers:
      - name: tools
        tools:
          - name: echo
            description: Echo input
            command: ["python3", "echo.py"]
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ext, ok := reg.Get("tool-pack")
	if !ok {
		t.Fatal("tool-pack not loaded")
	}
	if len(ext.CustomMCPServers) != 1 {
		t.Fatalf("custom mcp: %+v", ext.CustomMCPServers)
	}
}

func TestExpandMCPServersCustomRef(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".loop", "extensions", "tool-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: loop.dev/extension/v1
name: tool-pack
version: 1.0.0
contributions:
  aiAssets:
    mcpServers:
      - name: tools
        tools:
          - name: echo
            command: ["python3", "echo.py"]
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	expanded, pending, err := reg.ExpandMCPServers([]model.ADLMCPServer{
		{Ref: "ext:tool-pack/tools"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 0 {
		t.Fatalf("expanded: %+v", expanded)
	}
	if len(pending) != 1 || pending[0].Server.Name != "tools" {
		t.Fatalf("pending: %+v", pending)
	}
}

func TestResolveSkillCustomRef(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".loop", "extensions", "skill-pack")
	skillDir := filepath.Join(extDir, "skills", "lint")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Lint\n"), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: loop.dev/extension/v1
name: skill-pack
version: 1.0.0
contributions:
  aiAssets:
    skills:
      - name: lint
        path: ./skills/lint
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	skill, dir, err := reg.ResolveSkill("ext:skill-pack/lint")
	if err != nil {
		t.Fatal(err)
	}
	if skill.Name != "lint" || dir == "" {
		t.Fatalf("skill: %+v dir=%q", skill, dir)
	}
}

func TestResolveRuleRef(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".loop", "extensions", "guide-pack")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "guidelines.md"), []byte("Always run tests before merging.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: loop.dev/extension/v1
name: guide-pack
version: 1.0.0
contributions:
  aiAssets:
    rules:
      - name: inline
        content: |
          Follow the style guide.
      - name: from-file
        path: ./guidelines.md
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	inline, err := reg.ResolveRule("ext:guide-pack/inline")
	if err != nil {
		t.Fatal(err)
	}
	if inline != "Follow the style guide." {
		t.Fatalf("inline: %q", inline)
	}
	fromFile, err := reg.ResolveRule("ext:guide-pack/from-file")
	if err != nil {
		t.Fatal(err)
	}
	if fromFile != "Always run tests before merging.\n" {
		t.Fatalf("from-file: %q", fromFile)
	}
}

func TestMaterializeCustomMCPServer(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	proxyPath := filepath.Join(repoRoot, "harness-sdk", "loop_mcp_tools.py")
	if _, err := os.Stat(proxyPath); err != nil {
		t.Skip("harness-sdk not found from test cwd")
	}
	t.Setenv("LOOP_MCP_TOOLS_PATH", proxyPath)

	configDir := t.TempDir()
	pending := extensions.PendingCustomMCPServer{
		ExtensionName: "tool-pack",
		ExtensionDir:  configDir,
		Server: extensions.ExtensionCustomMCPServer{
			Name: "tools",
			Tools: []extensions.ExtensionMCPTool{
				{Name: "echo", Description: "Echo", Command: []string{"python3", "echo.py"}},
			},
		},
	}
	srv, err := extensions.MaterializeCustomMCPServer(configDir, pending)
	if err != nil {
		t.Fatal(err)
	}
	if srv.Name != "ext-tool-pack-tools" || srv.Type != "stdio" {
		t.Fatalf("server: %+v", srv)
	}
	if len(srv.Args) != 2 {
		t.Fatalf("args: %v", srv.Args)
	}
	cfgData, err := os.ReadFile(srv.Args[1])
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		ServerName string `json:"serverName"`
		Tools      []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "tools" || len(cfg.Tools) != 1 || cfg.Tools[0].Name != "echo" {
		t.Fatalf("cfg: %+v", cfg)
	}
}
