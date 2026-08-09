// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestCoordinatorDeleteSession(t *testing.T) {
	withTempHome(t)
	c := NewCoordinator(nil)
	keep, err := c.Create(context.Background(), CreateInput{
		SessionID: "sess-keep",
		Payload:   map[string]any{"message": "keep"},
	})
	if err != nil {
		t.Fatal(err)
	}
	drop, err := c.Create(context.Background(), CreateInput{
		SessionID: "sess-drop",
		Payload:   map[string]any{"message": "drop"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Respond(context.Background(), drop.RequestID, RespondInput{
		Answers: map[string]any{"ok": true},
	}); err != nil {
		t.Fatal(err)
	}

	if err := c.DeleteSession("sess-drop"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(context.Background(), drop.RequestID); err != ErrNotFound {
		t.Fatalf("deleted request err = %v", err)
	}
	if _, err := c.Get(context.Background(), keep.RequestID); err != nil {
		t.Fatalf("kept request err = %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".nui", "hitl-requests.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sess-drop") {
		t.Fatalf("disk still contains deleted session: %s", raw)
	}
	if !strings.Contains(string(raw), "sess-keep") {
		t.Fatalf("disk missing kept session: %s", raw)
	}
}
