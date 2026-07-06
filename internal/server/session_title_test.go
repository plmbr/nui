// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"loop/internal/agent"
	"loop/internal/model"
)

func TestShouldAutoTitle(t *testing.T) {
	def := model.ADLDefinition{ID: "claude-code", Name: "Claude Code"}
	session := model.Session{Name: PendingSessionTitle, AgentType: "claude-code"}
	msgs := []model.ChatMessage{{Role: "user", Content: "hello"}}

	if !shouldAutoTitle(session, def, msgs) {
		t.Fatal("expected auto title for pending name and first user message")
	}

	session.Name = "Claude Code"
	if !shouldAutoTitle(session, def, msgs) {
		t.Fatal("expected auto title for legacy agent label placeholder")
	}

	session.Name = "Custom"
	if shouldAutoTitle(session, def, msgs) {
		t.Fatal("expected skip when user set a custom name")
	}

	session.Name = "Claude Code"
	session.ScheduleID = "sched-1"
	if shouldAutoTitle(session, def, msgs) {
		t.Fatal("expected skip for scheduled sessions")
	}

	session.ScheduleID = ""
	msgs = append(msgs, model.ChatMessage{Role: "user", Content: "second"})
	if shouldAutoTitle(session, def, msgs) {
		t.Fatal("expected skip after second user message")
	}
}

func TestFallbackSessionTitle(t *testing.T) {
	got := fallbackSessionTitle([]model.ChatMessage{
		{Role: "user", Content: "How do I add auth to my API?"},
	})
	if got != "How do I add auth to my API?" {
		t.Fatalf("fallback = %q", got)
	}
}

func TestSanitizeSessionTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`  "Fix login bug"  `, "Fix login bug"},
		{"Line one\nLine two", "Line one"},
		{"", ""},
		{"   ", ""},
	}
	for _, tc := range tests {
		if got := sanitizeSessionTitle(tc.in); got != tc.want {
			t.Fatalf("sanitizeSessionTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildTitlePrompt(t *testing.T) {
	msgs := []model.ChatMessage{
		{Role: "user", Content: "How do I add auth?"},
		{Role: "assistant", Content: "Use middleware."},
	}
	prompt := buildTitlePrompt(msgs)
	if !strings.Contains(prompt, "User: How do I add auth?") {
		t.Fatalf("prompt missing user line: %q", prompt)
	}
	if !strings.Contains(prompt, "Assistant: Use middleware.") {
		t.Fatalf("prompt missing assistant line: %q", prompt)
	}
}

func TestMaybeGenerateSessionTitle_renamesDefaultSession(t *testing.T) {
	mgr := setupTestServerEnv(t)
	titleCalls := 0
	mgr.SetTestHarnessRun(func(_ context.Context, req agent.RunRequest, events chan<- agent.Event) error {
		if req.Ephemeral {
			titleCalls++
			if req.SystemPrompt != sessionTitleSystemPrompt {
				t.Fatalf("system prompt = %q", req.SystemPrompt)
			}
			events <- agent.Event{Type: agent.EventText, Content: "Add Auth Middleware"}
			events <- agent.Event{Type: agent.EventDone, SessionID: "ephemeral-session"}
			return nil
		}
		events <- agent.Event{Type: agent.EventText, Content: "stub reply"}
		events <- agent.Event{Type: agent.EventDone, SessionID: "stub-session-id"}
		return nil
	})

	session := seedSession("sess-title", PendingSessionTitle, "claude-code", t.TempDir())
	mu.Lock()
	sessionMessages[session.ID] = []model.ChatMessage{
		{Role: "user", Content: "How do I add auth to my API?"},
		{Role: "assistant", Content: "Use middleware."},
	}
	mu.Unlock()

	maybeGenerateSessionTitle(session.ID)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.RLock()
		s, ok := findSession(session.ID)
		mu.RUnlock()
		if ok && s.Name == "Add Auth Middleware" {
			if titleCalls != 1 {
				t.Fatalf("title harness calls = %d, want 1", titleCalls)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.RLock()
	s, _ := findSession(session.ID)
	agentID := agentSessions[session.ID]
	mu.RUnlock()
	if s.Name != "Add Auth Middleware" {
		t.Fatalf("session name = %q, want %q", s.Name, "Add Auth Middleware")
	}
	if agentID != "" {
		t.Fatalf("agentSessions should be unchanged, got %q", agentID)
	}
}

func TestHandleSessionAGUI_autoTitlesDefaultSession(t *testing.T) {
	mgr := setupTestServerEnv(t)
	titleCalls := 0
	mgr.SetTestHarnessRun(func(_ context.Context, req agent.RunRequest, events chan<- agent.Event) error {
		if req.Ephemeral {
			titleCalls++
			events <- agent.Event{Type: agent.EventText, Content: "Review README"}
			events <- agent.Event{Type: agent.EventDone, SessionID: "title-session"}
			return nil
		}
		events <- agent.Event{Type: agent.EventText, Content: "Done reviewing."}
		events <- agent.Event{Type: agent.EventDone, SessionID: "main-session"}
		return nil
	})

	session := seedSession("sess-agui-title", PendingSessionTitle, "claude-code", t.TempDir())

	code, events := postAGUI(t, session.ID, "Review the README please")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	var sawFinished bool
	for _, ev := range events {
		if ev.Type == "RUN_FINISHED" {
			sawFinished = true
		}
	}
	if !sawFinished {
		t.Fatal("expected RUN_FINISHED")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.RLock()
		s, ok := findSession(session.ID)
		mu.RUnlock()
		if ok && s.Name == "Review README" {
			if titleCalls != 1 {
				t.Fatalf("title harness calls = %d, want 1", titleCalls)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.RLock()
	s, _ := findSession(session.ID)
	mu.RUnlock()
	t.Fatalf("session name = %q, want %q (titleCalls=%d)", s.Name, "Review README", titleCalls)
}

func TestMaybeGenerateSessionTitle_usesFallbackWhenHarnessEmpty(t *testing.T) {
	setupTestServerEnv(t)
	session := seedSession("sess-fallback", PendingSessionTitle, "claude-code", t.TempDir())
	mu.Lock()
	sessionMessages[session.ID] = []model.ChatMessage{
		{Role: "user", Content: "Explain bubble sort briefly"},
	}
	mu.Unlock()

	maybeGenerateSessionTitle(session.ID)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.RLock()
		s, ok := findSession(session.ID)
		mu.RUnlock()
		if ok && s.Name == "Explain bubble sort briefly" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.RLock()
	s, _ := findSession(session.ID)
	mu.RUnlock()
	t.Fatalf("session name = %q, want fallback from user message", s.Name)
}
