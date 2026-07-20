// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func withTempHome(t *testing.T) {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
}

func TestCoordinatorCreateAndRespond(t *testing.T) {
	withTempHome(t)
	c := NewCoordinator(nil)
	req, err := c.Create(context.Background(), CreateInput{
		SessionID: "sess-1",
		Kind:      KindQuestion,
		Payload:   map[string]any{"message": "Pick one"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.RequestID == "" || req.Status != StatusDelivered {
		t.Fatalf("req = %+v", req)
	}

	resp, err := c.Respond(context.Background(), req.RequestID, RespondInput{
		Answers: map[string]any{"choice": "a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != StatusAnswered {
		t.Fatalf("resp = %+v", resp)
	}

	_, err = c.Respond(context.Background(), req.RequestID, RespondInput{})
	if err != ErrAlreadyResolved {
		t.Fatalf("err = %v", err)
	}
}

func TestCoordinatorWaitReceivesResponse(t *testing.T) {
	withTempHome(t)
	c := NewCoordinator(nil)
	req, err := c.Create(context.Background(), CreateInput{
		SessionID: "sess-1",
		Payload:   map[string]any{"message": "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Respond(context.Background(), req.RequestID, RespondInput{
		Answers: map[string]any{"ok": true},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.Wait(context.Background(), req.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || resp.Status != StatusAnswered {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestCoordinatorDisabledWhenPolicyOff(t *testing.T) {
	withTempHome(t)
	c := NewCoordinator(nil)
	c.SetPolicyFn(func(sessionID, runID string) (string, bool) {
		return ModeOff, true
	})
	_, err := c.Create(context.Background(), CreateInput{
		SessionID: "sess-1",
		Payload:   map[string]any{"message": "blocked"},
	})
	if err != ErrHitlDisabled {
		t.Fatalf("err = %v", err)
	}
}

func TestCoordinatorDuplicateRequestID(t *testing.T) {
	withTempHome(t)
	c := NewCoordinator(nil)
	in := CreateInput{
		RequestID: "fixed-id",
		SessionID: "sess-1",
		Payload:   map[string]any{"message": "first"},
	}
	first, err := c.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if first.RequestID != second.RequestID {
		t.Fatalf("ids differ: %q vs %q", first.RequestID, second.RequestID)
	}

	_, err = c.Respond(context.Background(), first.RequestID, RespondInput{
		Answers: map[string]any{"done": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Create(context.Background(), in)
	if err != ErrDuplicateRequest {
		t.Fatalf("err = %v", err)
	}
}

func TestCoordinatorCancel(t *testing.T) {
	withTempHome(t)
	c := NewCoordinator(nil)
	req, err := c.Create(context.Background(), CreateInput{
		SessionID: "sess-1",
		Payload:   map[string]any{"message": "cancel me"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Cancel(context.Background(), req.RequestID, "user dismissed"); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(context.Background(), req.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusCancelled {
		t.Fatalf("status = %q", got.Status)
	}
}
