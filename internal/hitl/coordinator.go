// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("hitl request not found")
	ErrAlreadyResolved   = errors.New("hitl request already resolved")
	ErrHitlDisabled      = errors.New("human in the loop is disabled for this session")
	ErrDuplicateRequest  = errors.New("hitl request id already exists")
)

// ListFilter selects stored requests.
type ListFilter struct {
	SessionID   string
	RunID       string
	Status      string
	PendingOnly bool
}

// DeliveryHook notifies channels when a request is created or resolved.
type DeliveryHook func(event string, req Request, resp *Response)

// Coordinator manages HITL request lifecycle.
type Coordinator struct {
	store    *persistedStore
	mu       sync.Mutex
	waiters  map[string][]chan *Response
	onEvent  DeliveryHook
	policyFn func(sessionID, runID string) (mode string, ok bool)
}

var defaultCoordinator *Coordinator

func Default() *Coordinator {
	if defaultCoordinator == nil {
		defaultCoordinator = NewCoordinator(nil)
	}
	return defaultCoordinator
}

func NewCoordinator(onEvent DeliveryHook) *Coordinator {
	return &Coordinator{
		store:   newStore(),
		waiters: map[string][]chan *Response{},
		onEvent: onEvent,
	}
}

func (c *Coordinator) SetPolicyFn(fn func(sessionID, runID string) (mode string, ok bool)) {
	c.policyFn = fn
}

func (c *Coordinator) Create(ctx context.Context, in CreateInput) (*Request, error) {
	if in.Kind == "" {
		in.Kind = KindQuestion
	}
	if in.TTLSeconds <= 0 {
		in.TTLSeconds = 3600
	}
	if len(in.Routing.Channels) == 0 {
		in.Routing.Channels = []string{ChannelLoopUI}
	}

	mode := ModeInteractive
	if c.policyFn != nil && in.SessionID != "" {
		if m, ok := c.policyFn(in.SessionID, in.RunID); ok {
			mode = m
		}
	}
	if mode == ModeOff || mode == ModeAuto {
		return nil, ErrHitlDisabled
	}

	in.Payload = NormalizePayload(in.Payload)

	id := in.RequestID
	if id == "" {
		id = uuid.NewString()
	}
	if existing, ok := c.store.getRequest(id); ok {
		if isPendingStatus(existing.Status) {
			copy := *existing
			return &copy, nil
		}
		return nil, ErrDuplicateRequest
	}

	now := time.Now().UTC()
	req := &Request{
		SchemaVersion: SchemaVersion,
		RequestID:     id,
		CorrelationID: in.CorrelationID,
		SessionID:     in.SessionID,
		RunID:         in.RunID,
		StepName:      in.StepName,
		Kind:          in.Kind,
		Routing:       in.Routing,
		Payload:       in.Payload,
		TTLSeconds:    in.TTLSeconds,
		Status:        StatusPending,
		CreatedAt:     now.Format(time.RFC3339),
		ExpiresAt:     now.Add(time.Duration(in.TTLSeconds) * time.Second).Format(time.RFC3339),
	}
	if err := c.store.putRequest(req); err != nil {
		return nil, err
	}

	req.Status = StatusDelivered
	_ = c.store.putRequest(req)

	if c.onEvent != nil {
		c.onEvent("created", *req, nil)
	}

	out := *req
	return &out, nil
}

// CreateOrchestrationGate creates a gate even when runtime hitl.mode is off (workflow steps).
func (c *Coordinator) CreateOrchestrationGate(ctx context.Context, in CreateInput) (*Request, error) {
	prev := c.policyFn
	c.policyFn = nil
	defer func() { c.policyFn = prev }()
	return c.Create(ctx, in)
}

func (c *Coordinator) Get(_ context.Context, requestID string) (*Request, error) {
	req, ok := c.store.getRequest(requestID)
	if !ok {
		return nil, ErrNotFound
	}
	return req, nil
}

func (c *Coordinator) ListPending(_ context.Context, filter ListFilter) ([]Request, error) {
	filter.PendingOnly = true
	return c.store.listRequests(filter), nil
}

func (c *Coordinator) Wait(ctx context.Context, requestID string) (*Response, error) {
	if resp, ok := c.store.getResponse(requestID); ok {
		return resp, nil
	}
	req, ok := c.store.getRequest(requestID)
	if !ok {
		return nil, ErrNotFound
	}
	if !isPendingStatus(req.Status) {
		if resp, ok := c.store.getResponse(requestID); ok {
			return resp, nil
		}
		return nil, ErrAlreadyResolved
	}

	ch := make(chan *Response, 1)
	c.mu.Lock()
	c.waiters[requestID] = append(c.waiters[requestID], ch)
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		waiters := c.waiters[requestID]
		for i, w := range waiters {
			if w == ch {
				c.waiters[requestID] = append(waiters[:i], waiters[i+1:]...)
				break
			}
		}
		if len(c.waiters[requestID]) == 0 {
			delete(c.waiters, requestID)
		}
		c.mu.Unlock()
	}()

	if req.ExpiresAt != "" {
		if exp, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
			timeout := time.Until(exp)
			if timeout > 0 {
				timer := time.NewTimer(timeout)
				defer timer.Stop()
				select {
				case resp := <-ch:
					return resp, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-timer.C:
					_, _ = c.expire(context.Background(), requestID)
					return nil, fmt.Errorf("hitl request expired")
				}
			}
		}
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Coordinator) Respond(_ context.Context, requestID string, in RespondInput) (*Response, error) {
	if _, ok := c.store.getResponse(requestID); ok {
		return nil, ErrAlreadyResolved
	}
	req, ok := c.store.getRequest(requestID)
	if !ok {
		return nil, ErrNotFound
	}
	if !isPendingStatus(req.Status) {
		return nil, ErrAlreadyResolved
	}

	status := in.Status
	if status == "" {
		status = StatusAnswered
	}
	now := time.Now().UTC()
	resp := &Response{
		SchemaVersion: SchemaVersion,
		RequestID:     requestID,
		CorrelationID: req.CorrelationID,
		Status:        status,
		Answers:       in.Answers,
		RespondedAt:   now.Format(time.RFC3339),
	}
	if in.RespondedBy.Channel != "" {
		resp.RespondedBy = &in.RespondedBy
	}

	req.Status = status
	_ = c.store.putRequest(req)
	_ = c.store.putResponse(resp)

	c.mu.Lock()
	waiters := c.waiters[requestID]
	delete(c.waiters, requestID)
	c.mu.Unlock()
	for _, ch := range waiters {
		select {
		case ch <- resp:
		default:
		}
	}

	if c.onEvent != nil {
		c.onEvent("resolved", *req, resp)
	}
	return resp, nil
}

func (c *Coordinator) Cancel(_ context.Context, requestID, reason string) error {
	in := RespondInput{
		Status:  StatusCancelled,
		Answers: map[string]any{"reason": reason},
		RespondedBy: RespondedBy{
			Channel: "system",
		},
	}
	_, err := c.Respond(context.Background(), requestID, in)
	return err
}

func (c *Coordinator) expire(_ context.Context, requestID string) (*Response, error) {
	in := RespondInput{
		Status: StatusExpired,
		RespondedBy: RespondedBy{
			Channel: "system",
		},
	}
	return c.Respond(context.Background(), requestID, in)
}
