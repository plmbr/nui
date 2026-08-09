// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

type Session struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	WorkingDir   string         `json:"workingDir"`
	AgentType    string         `json:"agentType"`
	AgentConfig  map[string]any `json:"agentConfig,omitempty"`
	CreatedAt    string         `json:"createdAt"`
	ScheduleID   string         `json:"scheduleId,omitempty"`
	ScheduleName string         `json:"scheduleName,omitempty"`
	LastRunAt        string         `json:"lastRunAt,omitempty"`
}

type ChatImage struct {
	ID        string `json:"id"`
	MediaType string `json:"mediaType"`
	Data      string `json:"data"`
}

type ChatMessagePart struct {
	Type              string         `json:"type"`
	ID                string         `json:"id,omitempty"`
	Content           string         `json:"content,omitempty"`
	ToolCallID        string         `json:"toolCallId,omitempty"`
	ToolName          string         `json:"toolName,omitempty"`
	ToolArgs          map[string]any `json:"toolArgs,omitempty"`
	ToolResult        any            `json:"toolResult,omitempty"`
	SubagentTrace     []ChatMessagePart `json:"subagentTrace,omitempty"`
	MCPAppResourceURI string         `json:"mcpAppResourceUri,omitempty"`
	MCPAppServerName  string         `json:"mcpAppServerName,omitempty"`
	MCPAppToolInput   map[string]any `json:"mcpAppToolInput,omitempty"`
	VisualizationHTML string         `json:"visualizationHtml,omitempty"`
	VisualizationTitle string        `json:"visualizationTitle,omitempty"`
}

type ChatMessage struct {
	ID        string            `json:"id"`
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	CreatedAt string            `json:"createdAt"`
	Parts     []ChatMessagePart `json:"parts,omitempty"`
	Images    []ChatImage       `json:"images,omitempty"`
	Error     bool              `json:"error,omitempty"`
}

// ── ADL (Agent Definition Language) ─────────────────────────────────────────

// ADLDefinition describes a single agent or multi-step pipeline.
type ADLDefinition struct {
	ADL          string         `yaml:"adl"          json:"adl"`
	ID           string         `yaml:"id"           json:"id"`
	Name         string         `yaml:"name"         json:"name"`
	Description  string         `yaml:"description"  json:"description,omitempty"`
	Tags         []string       `yaml:"tags"         json:"tags,omitempty"`
	Version      string         `yaml:"version"      json:"version,omitempty"`
	Harness      ADLHarness     `yaml:"harness"      json:"harness"`
	AllowedHarnesses []string   `yaml:"allowedHarnesses" json:"allowedHarnesses,omitempty"` // optional CLI whitelist; omit = any CLI harness when default is CLI
	SystemPrompt string         `yaml:"systemPrompt" json:"systemPrompt,omitempty"`
	Skill        string         `yaml:"skill"        json:"skill,omitempty"`
	AIAssets     ADLAIAssets    `yaml:"aiAssets"     json:"aiAssets,omitempty"`
	Env          map[string]string `yaml:"env"          json:"env,omitempty"`
	PromptMode        string               `yaml:"promptMode"        json:"promptMode,omitempty"`        // user | auto; default user
	DefaultPrompt     string               `yaml:"defaultPrompt"     json:"defaultPrompt,omitempty"`     // auto mode when no launch prompt
	PromptSuggestions []ADLPromptSuggestion `yaml:"promptSuggestions" json:"promptSuggestions,omitempty"` // quick-start pills in chat UI
	WorkingDirInput   bool                 `yaml:"workingDirInput"   json:"workingDirInput,omitempty"`   // true = user picks working dir at session create
	HITL              ADLHITL              `yaml:"hitl"              json:"hitl,omitempty"`
	ToolApprovals     ADLToolApprovals     `yaml:"toolApprovals"     json:"toolApprovals,omitempty"`
	Evals             []ADLEval            `yaml:"evals"             json:"evals,omitempty"`
	SubAgents         []string             `yaml:"subAgents"         json:"subAgents,omitempty"` // orchestrator: registry agent IDs only
	Steps             []ADLStep            `yaml:"steps"             json:"steps,omitempty"`
	Internal          bool                 `yaml:"internal"          json:"internal,omitempty"` // hidden from UI/CLI discovery
}

// ADLPromptSuggestion is a quick-start prompt shown as a pill above the chat input.
type ADLPromptSuggestion struct {
	Title  string `yaml:"title"  json:"title"`
	Prompt string `yaml:"prompt" json:"prompt"`
	Icon   string `yaml:"icon"   json:"icon,omitempty"` // lucide icon name, e.g. sparkles
}

// ADLHarness specifies how a step executes.
//
// Step harness types (harness.type):
//   - "claude-code" — runs the claude CLI as a host subprocess
//   - "pi"          — runs the pi CLI as a host subprocess
//   - "codex"       — runs the codex CLI as a host subprocess
//   - "opencode"    — runs the opencode CLI as a host subprocess
//   - "docker"        — connects to an HTTP/SSE agent in a Docker container (requires image + containerPort)
//   - "devcontainer"  — nui-managed dev container sandbox (requires innerHarness)
//   - "remote"        — connects to a pre-running HTTP/SSE agent over the network (requires host + port)
//   - "api"           — in-process LLM API harness (anthropic, openai, gemini, ollama, openrouter, …)
//
// Sandbox options (harness.sandbox) — applies to "claude-code", "pi", "codex", and "opencode" harnesses:
//   - "none"        — run directly on the host (default)
//   - "bubblewrap"  — wrap the subprocess in a bubblewrap sandbox (Linux only)
//   - "docker"      — run the subprocess agent inside a Docker container
type ADLHarness struct {
	Type          string `yaml:"type"          json:"type"`
	Provider      string `yaml:"provider"      json:"provider,omitempty"` // harness type=api: anthropic | openai | gemini | ollama | openrouter
	Model         string `yaml:"model"         json:"model,omitempty"`
	BaseURL       string `yaml:"baseUrl"       json:"baseUrl,omitempty"`       // harness type=api: optional API base URL override
	APIKeyEnv     string `yaml:"apiKeyEnv"     json:"apiKeyEnv,omitempty"`     // harness type=api: env var for API key (default per provider)
	InnerHarness  string `yaml:"innerHarness"  json:"innerHarness,omitempty"`  // harness type=devcontainer: claude-code | pi | codex | opencode
	Sandbox       string `yaml:"sandbox"       json:"sandbox,omitempty"`       // "none" | "bubblewrap" | "docker"
	Image         string `yaml:"image"         json:"image,omitempty"`          // Docker image (sandbox=docker, harness type=docker, or devcontainer override)
	ContainerPort int    `yaml:"containerPort" json:"containerPort,omitempty"` // harness type=docker only
	Host          string            `yaml:"host"          json:"host,omitempty"`           // harness type=remote only
	Port          int               `yaml:"port"          json:"port,omitempty"`           // harness type=remote only
	Env           map[string]string `yaml:"env"           json:"env,omitempty"`
	Permissions   string            `yaml:"permissions"   json:"permissions,omitempty"` // interactive | bypass (claude-code, codex)
	DisableTools  bool              `yaml:"disableTools"  json:"disableTools,omitempty"` // harness type=api: omit tools/tool_choice from LLM requests
}

type ADLAIAssets struct {
	MCPServers       []ADLMCPServer       `yaml:"mcpServers"       json:"mcpServers,omitempty"`
	Skills           []ADLSkill           `yaml:"skills"           json:"skills,omitempty"`
	Rules            []ADLRule            `yaml:"rules"            json:"rules,omitempty"`
	MentionProviders []ADLMentionProvider `yaml:"mentionProviders" json:"mentionProviders,omitempty"`
}

// ADLRule references a harness rule file by ref or inline path/content.
type ADLRule struct {
	Name    string `yaml:"name"              json:"name"`
	Ref     string `yaml:"ref,omitempty"     json:"ref,omitempty"` // ext:<extension>/<rule-name>
	Path    string `yaml:"path,omitempty"    json:"path,omitempty"`
	Content string `yaml:"content,omitempty" json:"content,omitempty"`
}

// ADLMentionProvider references an extension mention provider for @-mention autocomplete.
type ADLMentionProvider struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Ref  string `yaml:"ref"            json:"ref"` // ext:<extension>/<provider-id>
}

// ADLSkill configures one agent skill under aiAssets.skills.
// Exactly one source: local path, ref, content, or git+path.
type ADLSkill struct {
	Name    string `yaml:"name"              json:"name"`
	Path    string `yaml:"path,omitempty"    json:"path,omitempty"`    // local dir/SKILL.md, or subpath within git repo
	Ref     string `yaml:"ref,omitempty"     json:"ref,omitempty"`     // named skill in ~/.nui/skills/
	Git     string `yaml:"git,omitempty"     json:"git,omitempty"`     // remote repo URL
	Version string `yaml:"version,omitempty" json:"version,omitempty"` // git tag/commit
	Content string `yaml:"content,omitempty" json:"content,omitempty"` // inline SKILL.md
}

// ADLMCPServerAuth holds optional OAuth client credentials for remote MCP servers.
type ADLMCPServerAuth struct {
	ClientID     string   `yaml:"clientId"     json:"clientId,omitempty"`
	ClientSecret string   `yaml:"clientSecret" json:"clientSecret,omitempty"`
	Scopes       []string `yaml:"scopes"       json:"scopes,omitempty"`
}

// ADLMCPServer configures one MCP server entry in aiAssets.mcpServers.
type ADLMCPServer struct {
	Name    string            `yaml:"name"              json:"name"`
	Ref     string            `yaml:"ref,omitempty"     json:"ref,omitempty"` // ext:<extension>/<server-name>
	URL     string            `yaml:"url"               json:"url,omitempty"`
	Command string            `yaml:"command"           json:"command,omitempty"`
	Args    []string          `yaml:"args"              json:"args,omitempty"`
	Type    string            `yaml:"type"              json:"type,omitempty"` // http | sse | stdio
	Env     map[string]string `yaml:"env,omitempty"     json:"env,omitempty"`     // stdio only
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"` // http | sse only
	Auth    *ADLMCPServerAuth `yaml:"auth,omitempty"    json:"auth,omitempty"`
}

type ADLOutput struct {
	Name string `yaml:"name" json:"name"`
	Type string `yaml:"type" json:"type"`
}

type ADLInput struct {
	From   string `yaml:"from"   json:"from"`
	As     string `yaml:"as"     json:"as,omitempty"`
	Filter string `yaml:"filter" json:"filter,omitempty"`
}

type ADLStep struct {
	Type         string       `yaml:"type"         json:"type,omitempty"` // agent (default) | hitl
	Name         string       `yaml:"name"         json:"name"`
	Harness      *ADLHarness  `yaml:"harness"      json:"harness,omitempty"`
	SystemPrompt string       `yaml:"systemPrompt" json:"systemPrompt,omitempty"`
	DependsOn    []string     `yaml:"dependsOn"    json:"dependsOn,omitempty"`
	AIAssets     ADLAIAssets  `yaml:"aiAssets"     json:"aiAssets,omitempty"`
	HITL         *ADLStepHITL `yaml:"hitl"         json:"hitl,omitempty"`
	Outputs      []ADLOutput  `yaml:"outputs"      json:"outputs,omitempty"`
	Inputs       []ADLInput   `yaml:"inputs"       json:"inputs,omitempty"`
}
