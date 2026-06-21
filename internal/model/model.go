// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

type Session struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	WorkingDir  string         `json:"workingDir"`
	AgentType   string         `json:"agentType"`
	AgentConfig map[string]any `json:"agentConfig,omitempty"`
	CreatedAt   string         `json:"createdAt"`
}

type ChatMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// ── ADL (Agent Definition Language) ─────────────────────────────────────────

// ADLDefinition describes a single agent or multi-step workflow.
// Kind "agent" (default) is selectable at session creation time.
// Kind "workflow" is an orchestration plan that sequences multiple steps.
type ADLDefinition struct {
	ADL          string         `yaml:"adl"          json:"adl"`
	Kind         string         `yaml:"kind"         json:"kind,omitempty"` // "agent" | "workflow"; defaults to "agent"
	Name         string         `yaml:"name"         json:"name"`
	Description  string         `yaml:"description"  json:"description,omitempty"`
	Version      string         `yaml:"version"      json:"version,omitempty"`
	Harness      ADLHarness     `yaml:"harness"      json:"harness"`
	SystemPrompt string         `yaml:"systemPrompt" json:"systemPrompt,omitempty"`
	Skill        string         `yaml:"skill"        json:"skill,omitempty"`
	AIAssets     ADLAIAssets    `yaml:"aiAssets"     json:"aiAssets,omitempty"`
	Env          map[string]string `yaml:"env"          json:"env,omitempty"`
	Steps        []ADLStep      `yaml:"steps"        json:"steps,omitempty"`
	Constraints  ADLConstraints `yaml:"constraints"  json:"constraints,omitempty"`
	Schedule     *ADLSchedule   `yaml:"schedule"     json:"schedule,omitempty"`
}

// ADLHarness specifies how a step executes.
//
// Step harness types (harness.type):
//   - "claude-code" — runs the claude CLI as a host subprocess
//   - "pi"          — runs the pi CLI as a host subprocess
//   - "codex"       — runs the codex CLI as a host subprocess
//   - "opencode"    — runs the opencode CLI as a host subprocess
//   - "docker"      — connects to an HTTP/SSE agent in a Docker container (requires image + containerPort)
//   - "remote"      — connects to a pre-running HTTP/SSE agent over the network (requires host + port)
//
// Sandbox options (harness.sandbox) — applies to "claude-code", "pi", "codex", and "opencode" harnesses:
//   - "none"        — run directly on the host (default)
//   - "bubblewrap"  — wrap the subprocess in a bubblewrap sandbox (Linux only)
//   - "docker"      — run the subprocess agent inside a Docker container
type ADLHarness struct {
	Type          string `yaml:"type"          json:"type"`
	Model         string `yaml:"model"         json:"model,omitempty"`
	Sandbox       string `yaml:"sandbox"       json:"sandbox,omitempty"`       // "none" | "bubblewrap" | "docker"
	WorkingDir    string `yaml:"workingDir"    json:"workingDir,omitempty"`
	Image         string `yaml:"image"         json:"image,omitempty"`          // Docker image (sandbox=docker or harness type=docker)
	ContainerPort int    `yaml:"containerPort" json:"containerPort,omitempty"` // harness type=docker only
	Host          string            `yaml:"host"          json:"host,omitempty"`           // harness type=remote only
	Port          int               `yaml:"port"          json:"port,omitempty"`           // harness type=remote only
	Env           map[string]string `yaml:"env"           json:"env,omitempty"`
}

type ADLAIAssets struct {
	MCPServers []ADLMCPServer `yaml:"mcpServers" json:"mcpServers,omitempty"`
}

// ADLMCPServer configures one MCP server entry in aiAssets.mcpServers.
type ADLMCPServer struct {
	Name    string   `yaml:"name"    json:"name"`
	URL     string   `yaml:"url"     json:"url,omitempty"`
	Command string   `yaml:"command" json:"command,omitempty"`
	Args    []string `yaml:"args"    json:"args,omitempty"`
	Type    string   `yaml:"type"    json:"type,omitempty"` // http | sse | stdio
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
	Name            string      `yaml:"name"            json:"name"`
	Policy          string      `yaml:"policy"          json:"policy,omitempty"`
	Harness         *ADLHarness `yaml:"harness"         json:"harness,omitempty"`
	SystemPrompt    string      `yaml:"systemPrompt"    json:"systemPrompt,omitempty"`
	DependsOn       []string    `yaml:"dependsOn"       json:"dependsOn,omitempty"`
	AIAssets        ADLAIAssets `yaml:"aiAssets"        json:"aiAssets,omitempty"`
	Outputs         []ADLOutput `yaml:"outputs"         json:"outputs,omitempty"`
	Inputs          []ADLInput  `yaml:"inputs"          json:"inputs,omitempty"`
	Approval        string      `yaml:"approval"        json:"approval,omitempty"`
	ApprovalTimeout string      `yaml:"approvalTimeout" json:"approvalTimeout,omitempty"`
}

type ADLConstraints struct {
	MaxTokens      int    `yaml:"maxTokens"      json:"maxTokens,omitempty"`
	Timeout        string `yaml:"timeout"        json:"timeout,omitempty"`
	Retries        int    `yaml:"retries"        json:"retries,omitempty"`
	MaxConcurrency int    `yaml:"maxConcurrency" json:"maxConcurrency,omitempty"`
}

type ADLSchedule struct {
	Cron     string `yaml:"cron"     json:"cron"`
	Timezone string `yaml:"timezone" json:"timezone,omitempty"`
}
