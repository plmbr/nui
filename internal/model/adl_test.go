// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestADLDefinitionYAML_aiAssets(t *testing.T) {
	raw := []byte(`adl: "1.0"
name: data-agent
harness:
  type: claude-code
  model: claude-sonnet-4-6
systemPrompt: |
  You are helpful.
aiAssets:
  mcpServers:
    - name: data-analytics-mcp-server
      url: http://localhost:9123/mcp
      type: http
`)

	var def ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}
	if def.Name != "data-agent" {
		t.Fatalf("name = %q", def.Name)
	}
	if len(def.AIAssets.MCPServers) != 1 {
		t.Fatalf("mcpServers: %v", def.AIAssets.MCPServers)
	}
	srv := def.AIAssets.MCPServers[0]
	if srv.Name != "data-analytics-mcp-server" || srv.URL != "http://localhost:9123/mcp" || srv.Type != "http" {
		t.Fatalf("server: %+v", srv)
	}
}
