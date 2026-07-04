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
	// AgentConfig is the Loop session override map (hitlMode, harnessPermissions, …).
	AgentConfig map[string]any
}

type Agent interface {
	Name() string
	Run(ctx context.Context, req RunRequest, events chan<- Event) error
}
