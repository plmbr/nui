// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"loop/internal/model"
)

func TestFlowOutcomeByID(t *testing.T) {
	setFlowOutcome("flow-1", FlowStatusPending, "https://example.com/mcp", "")
	outcome, ok := FlowOutcomeByID("flow-1")
	if !ok || outcome.Status != FlowStatusPending {
		t.Fatalf("outcome = %+v, ok = %v", outcome, ok)
	}

	setFlowOutcome("flow-1", FlowStatusCompleted, "https://example.com/mcp", "")
	outcome, ok = FlowOutcomeByID("flow-1")
	if !ok || outcome.Status != FlowStatusCompleted {
		t.Fatalf("completed outcome = %+v, ok = %v", outcome, ok)
	}
}

func TestFlowSurvivesRequestContextCancel(t *testing.T) {
	flow := newPendingFlow(model.ADLMCPServer{Name: "test", URL: "https://example.com/mcp"}, "http://127.0.0.1:8080/callback")
	registerFlow(flow)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	flowCtx := context.WithoutCancel(ctx)
	select {
	case <-flowCtx.Done():
		t.Fatal("flow context should not be cancelled when request context is")
	default:
	}

	state := "TESTSTATE123"
	BindFlowState(state, flow.ID)

	got, ok := flowByState(state)
	if !ok {
		t.Fatal("flow should remain addressable by oauth state after request context cancel")
	}
	if got.ID != flow.ID {
		t.Fatalf("flow id = %q, want %q", got.ID, flow.ID)
	}

	select {
	case flow.resultCh <- nil:
	default:
	}

	removeFlow(flow.ID)
}

func TestBindFlowStateInFetcher(t *testing.T) {
	flow := newPendingFlow(model.ADLMCPServer{Name: "test", URL: "https://example.com/mcp"}, "http://127.0.0.1:8080/callback")
	registerFlow(flow)
	defer removeFlow(flow.ID)

	fetcher := fetcherForFlow(flow)
	authURL := "https://auth.example.com/authorize?state=FETCHERSTATE&client_id=abc"

	go func() {
		time.Sleep(10 * time.Millisecond)
		_, _ = fetcher(context.Background(), &auth.AuthorizationArgs{URL: authURL})
	}()

	select {
	case url := <-flow.authURLCh:
		if url != authURL {
			t.Fatalf("auth url = %q, want %q", url, authURL)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for auth url")
	}

	if _, ok := flowByState("FETCHERSTATE"); !ok {
		t.Fatal("fetcher should bind oauth state before returning auth url to caller")
	}
}
