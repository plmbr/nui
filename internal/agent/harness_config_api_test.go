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
	if !strings.Contains(prompt, "## Available skills") {
		t.Fatalf("prompt missing skills catalog: %q", prompt)
	}
	if !strings.Contains(prompt, "create-agent") {
		t.Fatal("expected create-agent skill in prompt")
	}
	if !strings.Contains(prompt, "load_skill") {
		t.Fatal("expected load_skill guidance in catalog")
	}
	// Progressive disclosure: full skill bodies are not inlined.
	if strings.Contains(prompt, "### Skill: create-agent") {
		t.Fatal("did not expect full skill body in API system prompt")
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

func TestExpandHarnessDeps_apiDisableToolsOmitsBuiltinMCP(t *testing.T) {
	expanded, err := ExpandHarnessDeps(HarnessDeps{}, nil, "api-session", model.ADLDefinition{
		Harness: model.ADLHarness{Type: "api", Provider: "openai", DisableTools: true},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, srv := range expanded.MCPServers {
		if srv.Name == "nui-viz" || srv.Name == nuiAgentMCPName ||
			srv.Name == nuiSkillsMCPName || srv.Name == nuiFSMCPName || srv.Name == nuiBashMCPName {
			t.Fatalf("unexpected builtin MCP when disableTools is set: %+v", srv)
		}
	}
	if strings.Contains(expanded.SystemPrompt, "show_visualization") {
		t.Fatal("disableTools api harness should not get viz system prompt")
	}
	if strings.Contains(expanded.SystemPrompt, "nui API tools") {
		t.Fatal("disableTools api harness should not get API tools system prompt")
	}
}

func TestExpandHarnessDeps_orchestratorIncludesCreateAgentSkillAndNuiAgentMCP(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	expanded, err := ExpandHarnessDeps(HarnessDeps{}, nil, "nui-session", model.ADLDefinition{
		ID:      "nui",
		Harness: model.ADLHarness{Type: "api", Provider: "anthropic"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	foundCreateAgent := false
	for _, skill := range expanded.Skills {
		if skill.Name == "create-agent" {
			foundCreateAgent = true
			break
		}
	}
	if !foundCreateAgent {
		t.Fatalf("skills = %+v, want create-agent", expanded.Skills)
	}
	foundOrchestrator := false
	foundAgent := false
	foundSkills := false
	foundFS := false
	foundBash := false
	for _, srv := range expanded.MCPServers {
		switch srv.Name {
		case nuiOrchestratorMCPName:
			foundOrchestrator = true
		case nuiAgentMCPName:
			foundAgent = true
		case nuiSkillsMCPName:
			foundSkills = true
		case nuiFSMCPName:
			foundFS = true
		case nuiBashMCPName:
			foundBash = true
		}
	}
	if !foundOrchestrator {
		t.Fatal("expected nui-orchestrator MCP")
	}
	if !foundAgent {
		t.Fatal("expected nui-agent MCP for save_agent")
	}
	if !foundSkills || !foundFS || !foundBash {
		t.Fatal("expected nui-skills/fs/bash MCP for orchestrator api harness")
	}
	prompt := assembleAPISystemPrompt(expanded)
	if !strings.Contains(prompt, "create-agent") {
		t.Fatalf("prompt missing create-agent skill: %q", prompt)
	}
	if !strings.Contains(prompt, "load_skill") {
		t.Fatal("expected progressive skill guidance in prompt")
	}
}

func TestExpandHarnessDeps_apiIncludesNuiAgentMCP(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	deps := HarnessDeps{}
	expanded, err := ExpandHarnessDeps(deps, nil, "api-session", model.ADLDefinition{
		Harness: model.ADLHarness{Type: "api", Provider: "anthropic"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	foundViz := false
	foundAgent := false
	foundSkills := false
	foundFS := false
	foundBash := false
	for _, srv := range expanded.MCPServers {
		switch srv.Name {
		case "nui-viz":
			foundViz = true
		case nuiAgentMCPName:
			foundAgent = true
		case nuiSkillsMCPName:
			foundSkills = true
			if srv.Env[envNuiSkillsRoot] == "" {
				t.Fatal("nui-skills missing NUI_SKILLS_ROOT")
			}
		case nuiFSMCPName:
			foundFS = true
		case nuiBashMCPName:
			foundBash = true
		}
	}
	if !foundViz {
		t.Fatal("expected nui-viz MCP")
	}
	if !foundAgent {
		t.Fatal("expected nui-agent MCP for api harness")
	}
	if !foundSkills || !foundFS || !foundBash {
		t.Fatal("expected nui-skills/fs/bash MCP for api harness")
	}
	if !strings.Contains(expanded.SystemPrompt, "nui API tools") {
		t.Fatal("expected API tools system prompt appendix")
	}
}

func TestExpandHarnessDeps_cliOmitsAPIWorkspaceMCPs(t *testing.T) {
	expanded, err := ExpandHarnessDeps(HarnessDeps{}, nil, "cli-session", model.ADLDefinition{
		Harness: model.ADLHarness{Type: "claude-code"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, srv := range expanded.MCPServers {
		switch srv.Name {
		case nuiSkillsMCPName, nuiFSMCPName, nuiBashMCPName:
			t.Fatalf("cli harness should not get %s", srv.Name)
		}
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
