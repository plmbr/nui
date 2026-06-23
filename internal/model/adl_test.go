// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestADLDefinitionYAML_aiAssets(t *testing.T) {
	raw := []byte(`adl: "1.0"
id: test-agent
name: Data Agent
harness:
  type: claude-code
  model: claude-sonnet-4-6
systemPrompt: |
  You are helpful.
aiAssets:
  mcpServers:
    - name: test-mcp-server
      url: http://localhost:3000/mcp
      type: http
      headers:
        Authorization: Bearer token
    - name: local-mcp
      command: npx
      args: ["-y", "pkg"]
      type: stdio
      env:
        API_KEY: secret
`)

	var def ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}
	if def.ID != "data-agent" {
		t.Fatalf("id = %q", def.ID)
	}
	if def.Name != "Data Agent" {
		t.Fatalf("name = %q", def.Name)
	}
	if len(def.AIAssets.MCPServers) != 2 {
		t.Fatalf("mcpServers: %v", def.AIAssets.MCPServers)
	}
	httpSrv := def.AIAssets.MCPServers[0]
	if httpSrv.Name != "test-mcp-server" || httpSrv.URL != "http://localhost:3000/mcp" || httpSrv.Type != "http" {
		t.Fatalf("http server: %+v", httpSrv)
	}
	if httpSrv.Headers["Authorization"] != "Bearer token" {
		t.Fatalf("http headers: %+v", httpSrv.Headers)
	}
	stdioSrv := def.AIAssets.MCPServers[1]
	if stdioSrv.Env["API_KEY"] != "secret" {
		t.Fatalf("stdio env: %+v", stdioSrv.Env)
	}
}

func TestADLDefinitionYAML_env(t *testing.T) {
	raw := []byte(`adl: "1.0"
id: env-agent
name: Env Agent
harness:
  type: claude-code
  model: claude-sonnet-4-6
  env:
    ANTHROPIC_API_KEY: harness-key
env:
  ANTHROPIC_BASE_URL: https://api.anthropic.com
`)

	var def ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}
	if def.Env["ANTHROPIC_BASE_URL"] != "https://api.anthropic.com" {
		t.Fatalf("global env: %v", def.Env)
	}
	if def.Harness.Env["ANTHROPIC_API_KEY"] != "harness-key" {
		t.Fatalf("harness env: %v", def.Harness.Env)
	}
}
