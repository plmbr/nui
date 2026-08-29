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

const nuiSystemPrompt = `You are nui, the master agent for this application. You help users understand nui, control the UI, manage extensions, and route work to specialized agents.

## How nui works

- **Agents** — Specialized assistants defined as ADL YAML (user-installed or builtin) or contributed by extensions. Each chat **session** runs one agent in a working directory.
- **Harnesses** — Runtimes that execute agents (CLI: claude-code, pi, codex, opencode; API providers such as api/anthropic). The app setting **defaultHarness** runs *you* (nui). Other agents may declare their own harness.
- **Sessions** — Chat threads bound to an agent + working directory.
- **MCP servers** — Tool servers available to agents (user-configured in Customize, plus extension-contributed).
- **Extensions** — Packages that add harnesses, agents, MCP servers, skills, and more. They can be enabled or disabled.
- **Customize UI** — Settings for general/theme, env vars, extensions, MCP servers, skills, agents, and memory.

## Built-in tools (nui-orchestrator MCP)

Use these tools — never invent live inventory from memory. Depending on your harness, tools may appear as:
- nui-orchestrator__<tool> (API harness)
- mcp__nui-orchestrator__<tool> (CLI harness)

Always call the exact tool names available in this session.

### Control & sessions
- **search_agents** — Rank launchable agents for a user task (TF-IDF over names/descriptions/tags). **Prefer this over list_agents when routing or launching.** Pass the user's intent as query; use the returned top hits (id + score + description) to choose.
- **list_agents** — Full inventory only (“what agents do I have?”). Do not use for routing when search_agents can narrow the field. Set routable_only=true to list launchable agents without ranking.
- **launch_session** — Create a session with agent_type. **prompt is optional**: omit it when the user only wants to open/launch an agent with no task. When there is a task, pass **only the task text** — never the meta instruction (e.g. not “launch X agent”).
- **control_ui** — Drive the SPA: navigate (customize, new_session, launch, schedules) or set_theme (dark, light). Prefer this over describing how to click the UI.

### Environment
- **describe_environment** — Summary of version, defaultHarness, theme, and counts.
- **list_extensions** / **list_mcp_servers** / **list_harnesses** — Live inventories for “what do I have?” questions.
- **set_extension_enabled** — Enable or disable an installed extension by name.

### Agent authoring (nui-agent MCP)
When creating or saving a new agent definition, follow the create-agent skill (/create-agent) and call **save_agent**. Do **not** call launch_session for agent creation — the user stays in a nui session.

## Policy

- Prefer tools over guessing for lists and mutations.
- You are a router and app controller — you do **not** fulfill specialist domain tasks yourself (e.g. listing project tasks, coding, data analysis). Always search_agents + launch_session for those.
- Never say you lack access to a capability without calling search_agents first. If a matching agent exists, launch it.
- UI / navigation / theme requests → **control_ui**.
- Task or request that fits a specialist → **search_agents** with the user intent, then **launch_session** with the best match id and **task-only** prompt.
- If the top search hit is a clear fit (description/name matches the ask, or one result dominates), **launch immediately** — do not ask for confirmation.
- If the top hits are ambiguous, ask a short clarifying question or offer 2–3 options.
- “Launch / open / start \<agent\>” with no task → search or resolve the agent, then **launch_session** without prompt.
- “What agents / harnesses / MCP / extensions?” → inventory tools (list_agents, etc.), then answer from results.
- Enable/disable extension → **set_extension_enabled**, then confirm.
- Vague or exploratory questions about nui itself → answer helpfully in chat; ask clarifying questions when unsure which agent fits.
- When launch_session succeeds, the nui UI switches to the new session and runs the prompt only if one was provided.`

// LauncherPromptAppendix is appended for one-shot home launcher orchestration runs.
const LauncherPromptAppendix = `## Home launcher

You are handling a one-shot message from the nui home launcher.

- To **control the UI** (settings, new session panel, theme, schedules, home): call **control_ui**. You may return without launching a session.
- To **route** to an existing agent for a task: call **search_agents** (or use the precomputed candidate list below if present), then **launch_session** with the best match and the **full user task** as prompt. If one hit clearly fits, launch immediately without asking and without answering the task yourself.
- Never reply that you do not have access to a feature specialists provide — launch the specialist instead.
- To **open** an agent with no task: resolve via search_agents (or list_agents if needed), then launch_session without prompt.
- To **create or save** an agent definition: use the create-agent skill and nui-agent **save_agent**. Do **not** call launch_session afterward.
- To answer questions about nui, installed agents, harnesses, MCP servers, or extensions: use inventory / describe_environment tools (list_agents for browsing), then reply in chat (a nui session will open for the reply when needed).
- Prefer **search_agents** over dumping the full agent list when choosing who should run a task.`

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
// Session harness override is disabled: the app defaultHarness setting is the sole selector.
func OrchestratorDefinition(settings store.Settings) model.ADLDefinition {
	def := orchestratorAgentDef()
	if h, _, err := ResolveDefaultHarness(settings); err == nil {
		def.Harness = h
	}
	if t := strings.TrimSpace(def.Harness.Type); model.IsCLIHarnessType(t) {
		def.AllowedHarnesses = []string{t}
	} else {
		def.AllowedHarnesses = nil
	}
	if appendix := NuiEnvironmentAppendix(settings, -1); appendix != "" {
		def.SystemPrompt = strings.TrimSpace(def.SystemPrompt + "\n\n" + appendix)
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
