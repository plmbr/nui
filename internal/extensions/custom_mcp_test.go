// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"loop/internal/extensions"
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
        install: true
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
	if !ext.CustomMCPServers[0].Install {
		t.Fatal("expected install: true")
	}
	installable := ext.InstallableCustomMCPServers()
	if len(installable) != 1 || installable[0].Name != "tools" {
		t.Fatalf("installable: %+v", installable)
	}
}

func TestMergeInstallableMCPServersForAgent(t *testing.T) {
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
        install: true
        tools:
          - name: echo
            command: ["python3", "echo.py"]
      - name: opt-in
        install: false
        tools:
          - name: hidden
            command: ["python3", "hidden.py"]
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	out := reg.MergeInstallableMCPServersForAgent(extensions.AgentHarnessDepsInput{}, "ext:tool-pack/reviewer")
	if len(out.PendingCustomMCPServers) != 1 {
		t.Fatalf("pending: %+v", out.PendingCustomMCPServers)
	}
	if out.PendingCustomMCPServers[0].Server.Name != "tools" {
		t.Fatalf("server name: %q", out.PendingCustomMCPServers[0].Server.Name)
	}
	out = reg.MergeInstallableMCPServersForAgent(extensions.AgentHarnessDepsInput{}, "adl:local-agent")
	if len(out.PendingCustomMCPServers) != 1 {
		t.Fatalf("install:true servers merge for all sessions: %+v", out.PendingCustomMCPServers)
	}
}

func TestMergeInstallableAIAssetsSkills(t *testing.T) {
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
        install: true
        path: ./skills/lint
      - name: optional
        install: false
        content: |
          ---
          name: optional
          ---
          Not installed.
`
	if err := os.WriteFile(filepath.Join(extDir, "extension.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	out, err := reg.MergeInstallableAIAssetsForAgent(extensions.AgentHarnessDepsInput{}, "adl:local-agent")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Skills) != 1 || out.Skills[0].Name != "lint" {
		t.Fatalf("skills: %+v", out.Skills)
	}
	if out.Skills[0].Path == "" {
		t.Fatalf("expected resolved path, got %+v", out.Skills[0])
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
