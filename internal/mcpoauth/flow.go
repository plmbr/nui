// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"nui/internal/model"
)

const flowTTL = 10 * time.Minute

// FlowStatus tracks interactive OAuth progress for the UI.
type FlowStatus string

const (
	FlowStatusPending   FlowStatus = "pending"
	FlowStatusCompleted FlowStatus = "completed"
	FlowStatusFailed    FlowStatus = "failed"
)

// FlowOutcome is the durable result of an OAuth flow (survives in-memory flow removal).
type FlowOutcome struct {
	FlowID    string     `json:"flowId"`
	ServerKey string     `json:"serverKey"`
	Status    FlowStatus `json:"status"`
	Error     string     `json:"error,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type pendingFlow struct {
	ID         string
	ServerKey  string
	Server     model.ADLMCPServer
	Redirect   string
	CreatedAt  time.Time
	authURLCh  chan string
	resultCh   chan *auth.AuthorizationResult
	errCh      chan error
	done       chan struct{}
}

var (
	flowMu       sync.Mutex
	flows        = map[string]*pendingFlow{}
	flowState    = map[string]string{} // oauth state -> flow ID
	flowOutcomes = map[string]FlowOutcome{}
)

func newPendingFlow(srv model.ADLMCPServer, redirect string) *pendingFlow {
	return &pendingFlow{
		ID:        uuid.NewString(),
		ServerKey: ServerKey(srv),
		Server:    srv,
		Redirect:  redirect,
		CreatedAt: time.Now().UTC(),
		authURLCh: make(chan string, 1),
		resultCh:  make(chan *auth.AuthorizationResult, 1),
		errCh:     make(chan error, 1),
		done:      make(chan struct{}),
	}
}

func registerFlow(f *pendingFlow) {
	flowMu.Lock()
	defer flowMu.Unlock()
	flows[f.ID] = f
	setFlowOutcomeLocked(f.ID, FlowStatusPending, f.ServerKey, "")
	pruneExpiredFlowsLocked()
}

func setFlowOutcome(flowID string, status FlowStatus, serverKey, errMsg string) {
	flowMu.Lock()
	defer flowMu.Unlock()
	setFlowOutcomeLocked(flowID, status, serverKey, errMsg)
}

func setFlowOutcomeLocked(flowID string, status FlowStatus, serverKey, errMsg string) {
	flowOutcomes[flowID] = FlowOutcome{
		FlowID:    flowID,
		ServerKey: serverKey,
		Status:    status,
		Error:     errMsg,
		UpdatedAt: time.Now().UTC(),
	}
	pruneFlowOutcomesLocked()
}

func pruneFlowOutcomesLocked() {
	cutoff := time.Now().Add(-flowTTL)
	for id, o := range flowOutcomes {
		if o.UpdatedAt.Before(cutoff) {
			delete(flowOutcomes, id)
		}
	}
}

// FlowOutcomeByID returns the latest status for an OAuth flow.
func FlowOutcomeByID(flowID string) (FlowOutcome, bool) {
	flowMu.Lock()
	defer flowMu.Unlock()
	o, ok := flowOutcomes[flowID]
	if !ok {
		return FlowOutcome{}, false
	}
	if time.Since(o.UpdatedAt) > flowTTL {
		delete(flowOutcomes, flowID)
		return FlowOutcome{}, false
	}
	return o, true
}

func pruneExpiredFlowsLocked() {
	cutoff := time.Now().Add(-flowTTL)
	for id, f := range flows {
		if f.CreatedAt.Before(cutoff) {
			delete(flows, id)
		}
	}
}

func flowByID(id string) (*pendingFlow, bool) {
	flowMu.Lock()
	defer flowMu.Unlock()
	f, ok := flows[id]
	if !ok {
		return nil, false
	}
	if time.Since(f.CreatedAt) > flowTTL {
		delete(flows, id)
		return nil, false
	}
	return f, true
}

func flowByState(state string) (*pendingFlow, bool) {
	flowMu.Lock()
	defer flowMu.Unlock()
	id, ok := flowState[state]
	if !ok {
		return nil, false
	}
	f, ok := flows[id]
	if !ok {
		delete(flowState, state)
		return nil, false
	}
	return f, true
}

func bindFlowState(state, flowID string) {
	flowMu.Lock()
	defer flowMu.Unlock()
	flowState[state] = flowID
}

func removeFlow(id string) {
	flowMu.Lock()
	defer flowMu.Unlock()
	delete(flows, id)
}
