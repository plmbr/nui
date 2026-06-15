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

type ADLDefinition struct {
	ADL          string         `yaml:"adl"          json:"adl"`
	Name         string         `yaml:"name"         json:"name"`
	Description  string         `yaml:"description"  json:"description,omitempty"`
	Version      string         `yaml:"version"      json:"version,omitempty"`
	Harness      ADLHarness     `yaml:"harness"      json:"harness"`
	SystemPrompt string         `yaml:"systemPrompt" json:"systemPrompt,omitempty"`
	Skill        string         `yaml:"skill"        json:"skill,omitempty"`
	Tools        ADLTools       `yaml:"tools"        json:"tools,omitempty"`
	Steps        []ADLStep      `yaml:"steps"        json:"steps,omitempty"`
	Constraints  ADLConstraints `yaml:"constraints"  json:"constraints,omitempty"`
	Schedule     *ADLSchedule   `yaml:"schedule"     json:"schedule,omitempty"`
}

type ADLHarness struct {
	Type          string `yaml:"type"          json:"type"`
	Model         string `yaml:"model"         json:"model,omitempty"`
	WorkingDir    string `yaml:"workingDir"    json:"workingDir,omitempty"`
	Image         string `yaml:"image"         json:"image,omitempty"`
	ContainerPort int    `yaml:"containerPort" json:"containerPort,omitempty"`
	Host          string `yaml:"host"          json:"host,omitempty"`
	Port          int    `yaml:"port"          json:"port,omitempty"`
}

type ADLTools struct {
	MCP []ADLMCPServer `yaml:"mcp" json:"mcp,omitempty"`
}

type ADLMCPServer struct {
	URL  string `yaml:"url"  json:"url"`
	Name string `yaml:"name" json:"name"`
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
	Tools           ADLTools    `yaml:"tools"           json:"tools,omitempty"`
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
