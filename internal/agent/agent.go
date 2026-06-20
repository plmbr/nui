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
)

type Event struct {
	Type       EventType `json:"type"`
	Content    string    `json:"content,omitempty"`
	SessionID  string    `json:"sessionId,omitempty"`
	Error      string    `json:"error,omitempty"`
	ToolCallID string    `json:"toolCallId,omitempty"`
	ToolName   string    `json:"toolName,omitempty"`
	ToolArgs   string    `json:"toolArgs,omitempty"`
}

type RunRequest struct {
	SessionID    string
	WorkingDir   string
	Message      string
	SystemPrompt string
}

type Agent interface {
	Name() string
	Run(ctx context.Context, req RunRequest, events chan<- Event) error
}
