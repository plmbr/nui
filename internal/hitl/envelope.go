// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

const SchemaVersion = "1"

const (
	KindQuestion  = "question"
	KindApproval  = "approval"
	KindFreeform  = "freeform"
	ChannelnuiUI = "nui-ui"
)

const (
	StatusPending   = "pending"
	StatusDelivered = "delivered"
	StatusAnswered  = "answered"
	StatusDeclined  = "declined"
	StatusCancelled = "cancelled"
	StatusExpired   = "expired"
)

// Mode values for ADL / session policy.
const (
	ModeInteractive = "interactive"
	ModeOff         = "off"
	ModeAuto        = "auto"
)

const (
	PermissionsInteractive = "interactive"
	PermissionsBypass      = "bypass"
)

// Request is the canonical HITL request envelope.
type Request struct {
	SchemaVersion string         `json:"schemaVersion"`
	RequestID     string         `json:"requestId"`
	CorrelationID string         `json:"correlationId,omitempty"`
	SessionID     string         `json:"sessionId,omitempty"`
	RunID         string         `json:"runId,omitempty"`
	StepName      string         `json:"stepName,omitempty"`
	Kind          string         `json:"kind"`
	Routing       Routing        `json:"routing,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
	TTLSeconds    int            `json:"ttlSeconds,omitempty"`
	Status        string         `json:"status"`
	CreatedAt     string         `json:"createdAt"`
	ExpiresAt     string         `json:"expiresAt,omitempty"`
}

type Routing struct {
	Channels []string `json:"channels,omitempty"`
}

// Response is the canonical HITL response envelope.
type Response struct {
	SchemaVersion string         `json:"schemaVersion"`
	RequestID     string         `json:"requestId"`
	CorrelationID string         `json:"correlationId,omitempty"`
	Status        string         `json:"status"`
	Answers       map[string]any `json:"answers,omitempty"`
	RespondedBy   *RespondedBy   `json:"respondedBy,omitempty"`
	RespondedAt   string         `json:"respondedAt,omitempty"`
}

type RespondedBy struct {
	Channel string `json:"channel"`
	Actor   string `json:"actor,omitempty"`
}

// CreateInput is the payload for creating a HITL request.
type CreateInput struct {
	RequestID     string
	CorrelationID string
	SessionID     string
	RunID         string
	StepName      string
	Kind          string
	Routing       Routing
	Payload       map[string]any
	TTLSeconds    int
}

// RespondInput is the payload for responding to a HITL request.
type RespondInput struct {
	Status      string
	Answers     map[string]any
	RespondedBy RespondedBy
}
