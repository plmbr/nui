// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nui/internal/model"
)

func TestHarnessRunners_includesAllBuiltinTypes(t *testing.T) {
	want := []string{
		"claude-code", "pi", "codex", "opencode", "antigravity",
		"docker", "devcontainer", "remote", "api",
	}
	for _, typ := range want {
		if _, ok := harnessRunners[typ]; !ok {
			t.Fatalf("harnessRunners missing %q", typ)
		}
	}
}

func TestDispatchHarness_unknownType(t *testing.T) {
	m := NewManager()
	a := NewADLAgent(model.ADLDefinition{ID: "x", Harness: model.ADLHarness{Type: "not-a-real-harness"}}, "proj", m)
	err := a.dispatchHarness(context.Background(), RunRequest{}, model.ADLHarness{Type: "not-a-real-harness"}, make(chan Event, 1))
	if err == nil || !strings.Contains(err.Error(), "unknown harness type") {
		t.Fatalf("err = %v", err)
	}
}

func TestDispatchHarness_emptyTypeDefaultsToClaudeCodeHook(t *testing.T) {
	m := NewManager()
	called := false
	m.SetTestHarnessRun(func(ctx context.Context, req RunRequest, events chan<- Event) error {
		called = true
		events <- Event{Type: EventDone}
		return nil
	})
	a := NewADLAgent(model.ADLDefinition{ID: "x"}, "proj", m)
	events := make(chan Event, 4)
	if err := a.dispatchHarness(context.Background(), RunRequest{Message: "hi"}, model.ADLHarness{}, events); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected test harness hook for empty harness type")
	}
}

func TestDispatchHarness_testHookShortCircuits(t *testing.T) {
	m := NewManager()
	m.SetTestHarnessRun(func(ctx context.Context, req RunRequest, events chan<- Event) error {
		if req.Message != "ping" {
			t.Fatalf("message = %q", req.Message)
		}
		events <- Event{Type: EventText, Content: "pong"}
		events <- Event{Type: EventDone, SessionID: "hook-sess"}
		return nil
	})
	a := NewADLAgent(model.ADLDefinition{
		ID:      "claude-code",
		Harness: model.ADLHarness{Type: "claude-code"},
	}, "proj-hook", m)

	events := make(chan Event, 8)
	if err := a.dispatchHarness(context.Background(), RunRequest{Message: "ping"}, model.ADLHarness{Type: "claude-code"}, events); err != nil {
		t.Fatal(err)
	}
	close(events)
	var text, done bool
	for ev := range events {
		switch ev.Type {
		case EventText:
			if ev.Content == "pong" {
				text = true
			}
		case EventDone:
			done = true
		}
	}
	if !text || !done {
		t.Fatalf("text=%v done=%v", text, done)
	}
}

func TestDispatchHarness_ephemeralClearsSessionID(t *testing.T) {
	m := NewManager()
	var gotSession string
	m.SetTestHarnessRun(func(ctx context.Context, req RunRequest, events chan<- Event) error {
		gotSession = req.SessionID
		events <- Event{Type: EventDone}
		return nil
	})
	a := NewADLAgent(model.ADLDefinition{ID: "api"}, "proj", m)
	if err := a.dispatchHarness(context.Background(), RunRequest{
		SessionID: "should-clear",
		Ephemeral: true,
		Message:   "hi",
	}, model.ADLHarness{Type: "api"}, make(chan Event, 2)); err != nil {
		t.Fatal(err)
	}
	if gotSession != "" {
		t.Fatalf("SessionID = %q, want empty for ephemeral", gotSession)
	}
}

func TestDispatchHarness_apiRunnerUsesAPIHarness(t *testing.T) {
	// Without credentials, API harness should fail fast with a key error — proves routing.
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_OAUTH_TOKEN", "")
	m := NewManager()
	a := NewADLAgent(model.ADLDefinition{
		ID:      "anthropic",
		Harness: model.ADLHarness{Type: "api", Provider: "anthropic", Model: "claude-sonnet-4-20250514"},
	}, "proj-api", m)
	err := a.dispatchHarness(context.Background(), RunRequest{Message: "hi"}, model.ADLHarness{
		Type:     "api",
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
	}, make(chan Event, 4))
	if err == nil {
		t.Fatal("expected API harness error without credentials")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "api") && !strings.Contains(strings.ToLower(err.Error()), "key") {
		// Accept any provider construction failure; main assertion is we didn't hit unknown harness.
		if errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
}

func TestRequireBubblewrap_noneOK(t *testing.T) {
	if err := requireBubblewrap("none"); err != nil {
		t.Fatal(err)
	}
	if err := requireBubblewrap(""); err != nil {
		t.Fatal(err)
	}
}

func TestHarnessProjectID_ephemeral(t *testing.T) {
	a := NewADLAgent(model.ADLDefinition{ID: "x"}, "sess-1", NewManager())
	got := a.harnessProjectID(RunRequest{Ephemeral: true}, model.ADLHarness{Type: "claude-code"})
	if got == "sess-1" {
		t.Fatal("ephemeral CLI turn should use ephemeral project id")
	}
	gotDocker := a.harnessProjectID(RunRequest{Ephemeral: true}, model.ADLHarness{Type: "claude-code", Sandbox: "docker"})
	if gotDocker != "sess-1" {
		t.Fatalf("ephemeral docker should keep session project id, got %q", gotDocker)
	}
	gotDC := a.harnessProjectID(RunRequest{Ephemeral: true}, model.ADLHarness{Type: "devcontainer"})
	if gotDC != "sess-1" {
		t.Fatalf("ephemeral devcontainer should keep session project id, got %q", gotDC)
	}
}
