// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"strings"

	"nui/internal/hitl"
	"nui/internal/model"
	"nui/internal/skills"
	"nui/internal/store"
)

// NuiAgentID is the built-in master agent for launcher routing and orchestration.
const NuiAgentID = "nui"

// OrchestratorAgentID is the legacy id alias for NuiAgentID.
const OrchestratorAgentID = NuiAgentID

const nuiSystemPrompt = `You are nui, the master agent for this workspace. You help users get work done by routing tasks to the best specialized agent when appropriate.

## Routing to another agent

When the user's request clearly fits a specific agent, use the **nui-orchestrator** MCP tools — do not route from memory or guess agent ids:
1. Call **list_agents** on the nui-orchestrator MCP server to discover available agent types.
2. Pick the best match from that list (use id, label, description, and tags).
3. Call **launch_session** with agent_type (exact id from list_agents) and the user's prompt.

Depending on your harness, these tools may appear as:
- nui-orchestrator__list_agents and nui-orchestrator__launch_session (API harness)
- mcp__nui-orchestrator__list_agents and mcp__nui-orchestrator__launch_session (CLI harness)

Always call the exact tool names available to you in this session.

When the request is vague, exploratory (e.g. "what can you do"), or you are unsure which agent fits, answer helpfully in chat and ask clarifying questions instead of guessing.
When the user wants to create or save a new agent definition, follow the create-agent skill (/create-agent) and call save_agent on the nui-agent MCP server. Do not call launch_session for agent creation — the user stays in a nui session.

In an ongoing session you can keep helping directly or delegate to another agent via launch_session when the user picks a direction. When launch_session succeeds, the nui UI switches to the new session and runs the provided prompt.`

// LauncherPromptAppendix is appended for one-shot home launcher orchestration runs.
const LauncherPromptAppendix = `## Home launcher

You are handling a one-shot message from the nui home launcher.

- To **route** the user to an existing agent: call nui-orchestrator **list_agents**, then **launch_session** with the chosen agent_type and prompt. Use the qualified tool names available to you (e.g. nui-orchestrator__list_agents or mcp__nui-orchestrator__list_agents).
- To **create or save** an agent definition: use the create-agent skill and nui-agent **save_agent**. Do **not** call launch_session afterward — the user should open a nui chat session.
- Call **launch_session** only when the user wants to **run a task now** with an existing agent (not when they are only defining a new agent).`

var nuiPromptSuggestions = []model.ADLPromptSuggestion{
	{
		Title:  "What can you do?",
		Icon:   "sparkles",
		Prompt: "What can you do? Which agents are available and when should I use each one?",
	},
	{
		Title:  "Pick an agent",
		Icon:   "route",
		Prompt: "Help me pick the best agent for my task. Ask what I'm trying to accomplish, then recommend one.",
	},
}

// IsNuiAgent reports whether id is the nui master agent.
func IsNuiAgent(id string) bool {
	id = strings.TrimSpace(id)
	return id == NuiAgentID || id == "nui-orchestrator"
}

// IsOrchestratorAgent reports whether id is the nui master agent.
func IsOrchestratorAgent(id string) bool {
	return IsNuiAgent(id)
}

// orchestratorAgentDef returns the base nui master agent ADL definition.
func orchestratorAgentDef() model.ADLDefinition {
	return model.ADLDefinition{
		ID:          NuiAgentID,
		Name:        "nui",
		Description: "The master agent — routes tasks to specialists or helps you explore what nui can do",
		Tags:        []string{"builtin", "nui"},
		Harness:     model.ADLHarness{Type: "api", Provider: "anthropic", Model: "claude-sonnet-4-20250514"},
		AIAssets: model.ADLAIAssets{
			Skills: []model.ADLSkill{skills.CreateAgentSkill()},
		},
		SystemPrompt:      nuiSystemPrompt,
		PromptSuggestions: nuiPromptSuggestions,
		HITL:              model.ADLHITL{Mode: hitl.ModeInteractive, Channels: []string{hitl.ChannelnuiUI}},
	}
}

// OrchestratorDefinition returns the nui agent def with harness resolved from settings.
func OrchestratorDefinition(settings store.Settings) model.ADLDefinition {
	def := orchestratorAgentDef()
	if h, _, err := ResolveDefaultHarness(settings); err == nil {
		def.Harness = h
	}
	return def
}

// InternalAgentDefs returns agents hidden from user discovery (none today).
func InternalAgentDefs() []model.ADLDefinition {
	return nil
}

// IsInternalAgent reports whether id names a hidden internal-only agent.
func IsInternalAgent(id string) bool {
	id = strings.TrimSpace(id)
	for _, def := range InternalAgentDefs() {
		if model.ADLAgentID(def) == id || def.ID == id {
			return true
		}
	}
	return false
}

// LookupInternalDefinition resolves a hidden internal agent by id.
func LookupInternalDefinition(agentType string) (model.ADLDefinition, bool) {
	agentType = strings.TrimSpace(agentType)
	for _, def := range InternalAgentDefs() {
		if model.ADLAgentID(def) == agentType || def.ID == agentType {
			return def, true
		}
	}
	return model.ADLDefinition{}, false
}

// IsOrchestratorRoutingTarget reports whether an agent id can be delegated to via nui tools.
func IsOrchestratorRoutingTarget(id string) bool {
	return !IsNuiAgent(id) && !IsInternalAgent(id)
}
