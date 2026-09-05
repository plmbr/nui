// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"

	"nui/internal/llm"
	"nui/internal/model"
)

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
	EventCouncilProgress EventType = "council_progress"
)

type Event struct {
	Type           EventType `json:"type"`
	Content        string    `json:"content,omitempty"`
	SessionID      string    `json:"sessionId,omitempty"`
	Error          string    `json:"error,omitempty"`
	ToolCallID     string    `json:"toolCallId,omitempty"`
	ToolName       string    `json:"toolName,omitempty"`
	ToolArgs       string    `json:"toolArgs,omitempty"`
	// ParentToolCallID scopes harness subagent output to a parent Task/Agent tool call.
	ParentToolCallID string `json:"parentToolCallId,omitempty"`
	ImageData      string    `json:"imageData,omitempty"`
	ImageMediaType string    `json:"imageMediaType,omitempty"`
	// Council carries progress for multi-agent deliberation (JSON payload in Content when Type is EventCouncilProgress).
	Council *CouncilProgress `json:"council,omitempty"`
}

// CouncilProgress is emitted during council rounds for UI progress strips.
type CouncilProgress struct {
	Phase            string `json:"phase"`                   // round_started | member_started | member_completed | member_failed | synthesizing | complete
	Round            string `json:"round,omitempty"`         // position | rebuttal | adjudication | synthesis
	RoundIndex       int    `json:"roundIndex,omitempty"`
	RoundsTotal      int    `json:"roundsTotal,omitempty"`
	MemberID         string `json:"memberId,omitempty"`
	MemberLabel      string `json:"memberLabel,omitempty"`
	MemberSessionID  string `json:"memberSessionId,omitempty"` // managed child nui session
	RunID            string `json:"runId,omitempty"`           // child run id for live attach
	MembersTotal     int    `json:"membersTotal,omitempty"`
	MembersDone      int    `json:"membersDone,omitempty"`
	Quorum           int    `json:"quorum,omitempty"`
	ElapsedMS        int64  `json:"elapsedMs,omitempty"`
	Error            string `json:"error,omitempty"`
	EstimatedCost    string `json:"estimatedCost,omitempty"` // rough estimate string for UI
}

// EphemeralAgentSuffix is appended to a nui session id for one-off harness runs that must
// not resume or share the main conversation's persistent agent instance.
const EphemeralAgentSuffix = "::ephemeral"

// EphemeralProjectID returns the Manager cache key for ephemeral harness runs.
func EphemeralProjectID(projectID string) string {
	return projectID + EphemeralAgentSuffix
}

type RunRequest struct {
	SessionID string
	// NuiSessionID is the nui session id for HITL/MCP env (distinct from harness resume SessionID).
	NuiSessionID string
	WorkingDir   string
	Message      string
	SystemPrompt string
	Model        string
	// ConfigDir is ~/.nui/sessions/<sessionID> with provisioned harness config.
	ConfigDir string
	// UserScopeHarness loads harness user/project settings via native CLI flags
	// instead of redirecting config through session-scoped env vars.
	UserScopeHarness bool
	// Env is merged ADL env (global + harness); applied to harness subprocesses.
	Env map[string]string
	// RunID is the active nui run when executing inside a tracked run.
	RunID string
	// HarnessPermissions is interactive | bypass for claude-code/codex native approval gates.
	HarnessPermissions string
	// ToolApprovalPolicy is default | all | allowlist | denylist for selective auto-approve.
	ToolApprovalPolicy string
	// ToolApprovalTools lists tool names/patterns for allowlist or denylist policies.
	ToolApprovalTools []string
	// AgentConfig is the nui session override map (hitlMode, harnessPermissions, …).
	AgentConfig map[string]any
	// Ephemeral runs use a separate harness agent instance and never resume SessionID.
	// Docker sandbox harnesses honor this via the HTTP "ephemeral" flag on a shared container.
	Ephemeral bool
	// ResolveADL looks up registry agent definitions (council members / step agents).
	ResolveADL ADLResolver
	// MemberHarnessSession returns the harness resume session id for a council member.
	MemberHarnessSession func(memberID string) string
	// OnMemberHarnessSession persists a council member harness session id after a run.
	OnMemberHarnessSession func(memberID, harnessSessionID string)
	// EnsureCouncilMemberSession creates or reuses a managed child nui session for a member.
	EnsureCouncilMemberSession func(memberID, label, agentType string) (childSessionID string, err error)
	// RunCouncilMemberSession starts a tracked run on a child session and waits for completion.
	// onStarted is invoked with the run id as soon as the run is registered (before wait).
	RunCouncilMemberSession func(ctx context.Context, childSessionID, message string, onStarted func(runID string)) (output, runID string, err error)
	// ExtraTools are native LLM tools merged into the api harness tool loop (not MCP).
	ExtraTools []llm.Tool
	// HandleExtraTool executes a native ExtraTools call. Return ok=false to fall through to MCP.
	HandleExtraTool func(ctx context.Context, name string, args map[string]any) (result string, ok bool, err error)
	// History is prior chat turns for harnesses that manage context in nui (api harness).
	History []model.ChatMessage
	// MCPServers are session-scoped MCP servers for the api harness tool loop.
	MCPServers []model.ADLMCPServer
	// APIProvider is the LLM provider id (from ADL harness.provider).
	APIProvider string
}

// ADLResolver resolves an ADL definition by canonical agent id.
type ADLResolver func(agentID string) (model.ADLDefinition, bool)

type Agent interface {
	Name() string
	Run(ctx context.Context, req RunRequest, events chan<- Event) error
}
