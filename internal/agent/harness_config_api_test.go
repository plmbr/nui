// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"

	"nui/internal/model"
)

func TestAssembleAPISystemPromptIncludesBuiltinSkills(t *testing.T) {
	prompt := assembleAPISystemPrompt(HarnessDeps{
		SystemPrompt: "Base prompt.",
		Skills:       nil,
	})
	if !strings.Contains(prompt, "## nui skills") {
		t.Fatalf("prompt missing skills section: %q", prompt)
	}
	if !strings.Contains(prompt, "create-agent") {
		t.Fatal("expected create-agent skill in prompt")
	}
	if !strings.Contains(prompt, "save_agent") {
		t.Fatal("expected save_agent instructions in prompt")
	}
	if !strings.Contains(prompt, "show_visualization") {
		t.Fatal("expected visualize skill in prompt")
	}
}

func TestExpandHarnessDeps_ollamaOmitsVisualizeSkill(t *testing.T) {
	expanded, err := ExpandHarnessDeps(HarnessDeps{}, nil, "ollama-session", model.ADLDefinition{
		Harness: model.ADLHarness{Type: "api", Provider: "ollama"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, skill := range expanded.Skills {
		if skill.Name == "visualize" {
			t.Fatal("ollama api harness should not include visualize skill")
		}
	}
	if strings.Contains(expanded.SystemPrompt, "Never end with \"building the chart\"") {
		t.Fatal("ollama should not get generic viz system prompt appendix")
	}
}

func TestExpandHarnessDeps_apiIncludesNuiAgentMCP(t *testing.T) {
	deps := HarnessDeps{}
	expanded, err := ExpandHarnessDeps(deps, nil, "api-session", model.ADLDefinition{
		Harness: model.ADLHarness{Type: "api", Provider: "anthropic"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	foundViz := false
	foundAgent := false
	for _, srv := range expanded.MCPServers {
		if srv.Name == "nui-viz" {
			foundViz = true
		}
		if srv.Name == nuiAgentMCPName {
			foundAgent = true
		}
	}
	if !foundViz {
		t.Fatal("expected nui-viz MCP")
	}
	if !foundAgent {
		t.Fatal("expected nui-agent MCP for api harness")
	}
}

func TestExpandHarnessDeps_cliIncludesNuiAgentMCP(t *testing.T) {
	deps := HarnessDeps{}
	expanded, err := ExpandHarnessDeps(deps, nil, "cli-session", model.ADLDefinition{
		ID:      "cli-agent",
		Harness: model.ADLHarness{Type: "claude-code"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, srv := range expanded.MCPServers {
		if srv.Name == nuiAgentMCPName {
			found = true
			if srv.Env["NUI_MEMORY_AGENT_ID"] != "cli-agent" {
				t.Fatalf("NUI_MEMORY_AGENT_ID = %q", srv.Env["NUI_MEMORY_AGENT_ID"])
			}
		}
	}
	if !found {
		t.Fatal("cli harness should include nui-agent MCP for memory updates")
	}
}
