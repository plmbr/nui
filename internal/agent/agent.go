// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "context"

type EventType string

const (
	EventText            EventType = "text"
	EventDone            EventType = "done"
	EventError           EventType = "error"
	EventToolCallStart   EventType = "tool_call_start"
	EventToolCallArgs    EventType = "tool_call_args"
	EventToolCallEnd     EventType = "tool_call_end"
	EventToolCallResult  EventType = "tool_call_result"
	EventImage           EventType = "image"
	EventHITLRequest     EventType = "hitl_request"
)

type Event struct {
	Type           EventType `json:"type"`
	Content        string    `json:"content,omitempty"`
	SessionID      string    `json:"sessionId,omitempty"`
	Error          string    `json:"error,omitempty"`
	ToolCallID     string    `json:"toolCallId,omitempty"`
	ToolName       string    `json:"toolName,omitempty"`
	ToolArgs       string    `json:"toolArgs,omitempty"`
	ImageData      string    `json:"imageData,omitempty"`
	ImageMediaType string    `json:"imageMediaType,omitempty"`
}

// EphemeralAgentSuffix is appended to a Loop session id for one-off harness runs that must
// not resume or share the main conversation's persistent agent instance.
const EphemeralAgentSuffix = "::ephemeral"

// EphemeralProjectID returns the Manager cache key for ephemeral harness runs.
func EphemeralProjectID(projectID string) string {
	return projectID + EphemeralAgentSuffix
}

type RunRequest struct {
	SessionID string
	// LoopSessionID is the Loop session id for HITL/MCP env (distinct from harness resume SessionID).
	LoopSessionID string
	WorkingDir   string
	Message      string
	SystemPrompt string
	Model        string
	// ConfigDir is ~/.loop/sessions/<sessionID> with provisioned harness config.
	ConfigDir string
	// UserScopeHarness loads harness user/project settings via native CLI flags
	// instead of redirecting config through session-scoped env vars.
	UserScopeHarness bool
	// Env is merged ADL env (global + harness); applied to harness subprocesses.
	Env map[string]string
	// RunID is the active Loop run when executing inside a tracked run.
	RunID string
	// HarnessPermissions is interactive | bypass for claude-code/codex native approval gates.
	HarnessPermissions string
	// ToolApprovalPolicy is default | all | allowlist | denylist for selective auto-approve.
	ToolApprovalPolicy string
	// ToolApprovalTools lists tool names/patterns for allowlist or denylist policies.
	ToolApprovalTools []string
	// AgentConfig is the Loop session override map (hitlMode, harnessPermissions, …).
	AgentConfig map[string]any
	// Ephemeral runs use a separate harness agent instance and never resume SessionID.
	// Docker sandbox harnesses honor this via the HTTP "ephemeral" flag on a shared container.
	Ephemeral bool
}

type Agent interface {
	Name() string
	Run(ctx context.Context, req RunRequest, events chan<- Event) error
}
