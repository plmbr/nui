// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"nui/internal/agent"
)

func TestHandleSessionAGUI_rejectsEmptyUserMessage(t *testing.T) {
	setupTestServerEnv(t)
	seedSession("sess-empty", "Test", testStubAgentType, t.TempDir())

	body := `{"messages":[{"id":"u1","role":"assistant","content":"hi"}]}`
	rec := postAGUIWithBody(t, "sess-empty", strings.NewReader(body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSessionAGUI_notFound(t *testing.T) {
	setupTestServerEnv(t)
	rec := postAGUIWithBody(t, "missing", strings.NewReader(`{"messages":[{"id":"u1","role":"user","content":"hi"}]}`))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleSessionAGUI_persistsSlashCommandNotExpandedSkill(t *testing.T) {
	mgr := setupTestServerEnv(t)
	var harnessMessage string
	mgr.SetTestHarnessRun(func(_ context.Context, req agent.RunRequest, events chan<- agent.Event) error {
		harnessMessage = req.Message
		events <- agent.Event{Type: agent.EventText, Content: "ok"}
		events <- agent.Event{Type: agent.EventDone, SessionID: "s1"}
		return nil
	})
	seedSession("sess-skill", "Test", testStubAgentType, t.TempDir())

	userInput := "/create-agent save as helper"
	code, _ := postAGUI(t, "sess-skill", userInput)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}

	mu.RLock()
	msgs := sessionMessages["sess-skill"]
	mu.RUnlock()
	if len(msgs) < 1 {
		t.Fatalf("expected persisted user message, got %d messages", len(msgs))
	}
	if msgs[0].Content != userInput {
		t.Fatalf("persisted user message = %q, want %q", msgs[0].Content, userInput)
	}
	if strings.HasPrefix(strings.TrimSpace(harnessMessage), "/create-agent") {
		t.Fatalf("harness received unexpanded message %q", harnessMessage)
	}
	if !strings.Contains(harnessMessage, "Create Agent") {
		t.Fatalf("harness message missing skill body: %q", harnessMessage)
	}
	if !strings.Contains(harnessMessage, "save as helper") {
		t.Fatalf("harness message missing user args: %q", harnessMessage)
	}
}

func TestHandleSessionAGUI_streamsEventsAndPersistsMessages(t *testing.T) {
	mgr := setupTestServerEnv(t)
	mgr.SetTestHarnessRun(stubHarnessRun("hello from stub"))
	seedSession("sess-agui", "Test", testStubAgentType, t.TempDir())

	code, events := postAGUI(t, "sess-agui", "ping")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}

	var sawStarted, sawText, sawFinished bool
	for _, ev := range events {
		switch ev.Type {
		case "RUN_STARTED":
			sawStarted = true
		case "TEXT_MESSAGE_CHUNK":
			if delta, _ := ev.Raw["delta"].(string); strings.Contains(delta, "hello from stub") {
				sawText = true
			}
		case "RUN_FINISHED":
			sawFinished = true
		}
	}
	if !sawStarted || !sawText || !sawFinished {
		t.Fatalf("events missing started/text/finished: %+v", events)
	}

	mu.RLock()
	msgs := sessionMessages["sess-agui"]
	mu.RUnlock()
	if len(msgs) < 2 {
		t.Fatalf("expected user + assistant messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "ping" {
		t.Fatalf("user message = %+v", msgs[0])
	}
}

func TestHandleSessionAGUI_serializesConcurrentRuns(t *testing.T) {
	mgr := setupTestServerEnv(t)
	var running sync.Mutex
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	mgr.SetTestHarnessRun(func(_ context.Context, req agent.RunRequest, events chan<- agent.Event) error {
		_ = req
		select {
		case entered <- struct{}{}:
		default:
		}
		running.Lock()
		defer running.Unlock()
		<-release
		events <- agent.Event{Type: agent.EventText, Content: "done"}
		events <- agent.Event{Type: agent.EventDone, SessionID: "s1"}
		return nil
	})
	seedSession("sess-lock", "Test", testStubAgentType, t.TempDir())

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		postAGUI(t, "sess-lock", "first")
	}()
	time.Sleep(50 * time.Millisecond)
	go func() {
		defer wg.Done()
		postAGUI(t, "sess-lock", "second")
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first AG-UI run to start")
	}
	close(release)
	wg.Wait()
}

func TestHandleSessionAGUI_methodNotAllowed(t *testing.T) {
	setupTestServerEnv(t)
	seedSession("sess-method", "Test", testStubAgentType, t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/sess-method/ag-ui", nil)
	rec := httptest.NewRecorder()
	handleSessionAGUI(rec, req, "sess-method")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandleSessionAGUI_invalidJSON(t *testing.T) {
	setupTestServerEnv(t)
	seedSession("sess-json", "Test", testStubAgentType, t.TempDir())
	rec := postAGUIWithBody(t, "sess-json", strings.NewReader(`{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleSessionAGUI_agentError(t *testing.T) {
	mgr := setupTestServerEnv(t)
	mgr.SetTestHarnessRun(func(_ context.Context, _ agent.RunRequest, events chan<- agent.Event) error {
		events <- agent.Event{Type: agent.EventError, Error: "boom"}
		return nil
	})
	seedSession("sess-err", "Test", testStubAgentType, t.TempDir())

	_, events := postAGUI(t, "sess-err", "fail")
	var sawError bool
	for _, ev := range events {
		if ev.Type == "RUN_ERROR" {
			sawError = true
		}
	}
	if !sawError {
		t.Fatalf("expected RUN_ERROR event, got %+v", events)
	}
}

func TestHandleSessionAGUI_visualizationCustomEvent(t *testing.T) {
	chartHTML := `<canvas id="c"></canvas><script>new Chart(document.getElementById("c"))</script>`
	toolArgsBytes, err := json.Marshal(map[string]string{"html": chartHTML, "title": "Chart"})
	if err != nil {
		t.Fatal(err)
	}
	toolArgs := string(toolArgsBytes)
	mgr := setupTestServerEnv(t)
	mgr.SetTestHarnessRun(func(_ context.Context, _ agent.RunRequest, events chan<- agent.Event) error {
		events <- agent.Event{
			Type:       agent.EventToolCallStart,
			ToolCallID: "tc-viz",
			ToolName:   "mcp__nui-viz__show_visualization",
		}
		events <- agent.Event{
			Type:       agent.EventToolCallArgs,
			ToolCallID: "tc-viz",
			ToolArgs:   toolArgs,
		}
		events <- agent.Event{
			Type:       agent.EventToolCallEnd,
			ToolCallID: "tc-viz",
			ToolName:   "mcp__nui-viz__show_visualization",
			ToolArgs:   toolArgs,
		}
		events <- agent.Event{Type: agent.EventDone, SessionID: "s1"}
		return nil
	})
	seedSession("sess-viz", "Test", testStubAgentType, t.TempDir())

	_, events := postAGUI(t, "sess-viz", "draw")
	var sawViz bool
	for _, ev := range events {
		if ev.Type == "CUSTOM" && ev.Raw["name"] == "visualization" {
			sawViz = true
			val, _ := ev.Raw["value"].(map[string]any)
			htmlVal, _ := val["html"].(string)
			if !strings.Contains(htmlVal, chartHTML) {
				t.Fatalf("viz html = %v", val["html"])
			}
		}
	}
	if !sawViz {
		t.Fatalf("expected visualization custom event, got %+v", events)
	}
}

func TestHandleSessionAGUI_subAgentRoutedCustomEvent(t *testing.T) {
	mgr := setupTestServerEnv(t)
	mgr.SetTestHarnessRun(func(_ context.Context, _ agent.RunRequest, events chan<- agent.Event) error {
		events <- agent.Event{
			Type:             agent.EventSubAgentRouted,
			RoutedAgentID:    "code-reviewer",
			RoutedAgentLabel: "Code Reviewer",
		}
		events <- agent.Event{Type: agent.EventText, Content: "Review complete."}
		events <- agent.Event{Type: agent.EventDone, SessionID: "sub-s1"}
		return nil
	})
	seedSession("sess-sub", "Test", testStubAgentType, t.TempDir())

	_, events := postAGUI(t, "sess-sub", "review my PR")
	var sawRouted bool
	for _, ev := range events {
		if ev.Type == "CUSTOM" && ev.Raw["name"] == "sub_agent_routed" {
			sawRouted = true
			val, _ := ev.Raw["value"].(map[string]any)
			if val["agentId"] != "code-reviewer" || val["label"] != "Code Reviewer" {
				t.Fatalf("routed value = %+v", val)
			}
		}
	}
	if !sawRouted {
		t.Fatalf("expected sub_agent_routed custom event, got %+v", events)
	}
}

func TestHandleSessionAGUI_openSessionCustomEvent(t *testing.T) {
	launchResult := `{
		"session": {"id":"target-sess","name":"Delegated","agentType":"claude-code","workingDir":"/tmp","createdAt":"2026-01-01T00:00:00Z"},
		"prompt": "fix the flaky test"
	}`
	mgr := setupTestServerEnv(t)
	mgr.SetTestHarnessRun(func(_ context.Context, _ agent.RunRequest, events chan<- agent.Event) error {
		events <- agent.Event{
			Type:       agent.EventToolCallStart,
			ToolCallID: "tc-launch",
			ToolName:   "nui-orchestrator__launch_session",
		}
		events <- agent.Event{
			Type:       agent.EventToolCallEnd,
			ToolCallID: "tc-launch",
			ToolName:   "nui-orchestrator__launch_session",
		}
		events <- agent.Event{
			Type:       agent.EventToolCallResult,
			ToolCallID: "tc-launch",
			ToolName:   "nui-orchestrator__launch_session",
			Content:    launchResult,
		}
		events <- agent.Event{Type: agent.EventDone, SessionID: "s1"}
		return nil
	})
	seedSession("sess-launch", "Test", testStubAgentType, t.TempDir())

	_, events := postAGUI(t, "sess-launch", "send this to claude")
	var sawOpen bool
	for _, ev := range events {
		if ev.Type == "CUSTOM" && ev.Raw["name"] == "open_session" {
			sawOpen = true
			val, _ := ev.Raw["value"].(map[string]any)
			if val["sessionId"] != "target-sess" {
				t.Fatalf("sessionId = %v", val["sessionId"])
			}
			if val["prompt"] != "fix the flaky test" {
				t.Fatalf("prompt = %v", val["prompt"])
			}
			if val["agentType"] != "claude-code" {
				t.Fatalf("agentType = %v", val["agentType"])
			}
		}
	}
	if !sawOpen {
		t.Fatalf("expected open_session custom event, got %+v", events)
	}
}

func TestHandleSessionAGUI_uiActionCustomEvent(t *testing.T) {
	uiResult := `{"ok":true,"actions":[{"type":"navigate","target":"customize"},{"type":"set_theme","theme":"dark"}]}`
	mgr := setupTestServerEnv(t)
	mgr.SetTestHarnessRun(func(_ context.Context, _ agent.RunRequest, events chan<- agent.Event) error {
		events <- agent.Event{
			Type:       agent.EventToolCallResult,
			ToolCallID: "tc-ui",
			ToolName:   "nui-orchestrator__control_ui",
			Content:    uiResult,
		}
		events <- agent.Event{Type: agent.EventDone, SessionID: "s1"}
		return nil
	})
	seedSession("sess-ui", "Test", testStubAgentType, t.TempDir())

	_, events := postAGUI(t, "sess-ui", "show settings")
	var saw bool
	for _, ev := range events {
		if ev.Type == "CUSTOM" && ev.Raw["name"] == "ui_action" {
			saw = true
			val, _ := ev.Raw["value"].(map[string]any)
			actions, _ := val["actions"].([]any)
			if len(actions) != 2 {
				t.Fatalf("actions = %+v", actions)
			}
		}
	}
	if !saw {
		t.Fatalf("expected ui_action custom event, got %+v", events)
	}
}

func TestHandleSessionAGUI_openSessionIgnoresFailedLaunch(t *testing.T) {
	mgr := setupTestServerEnv(t)
	mgr.SetTestHarnessRun(func(_ context.Context, _ agent.RunRequest, events chan<- agent.Event) error {
		events <- agent.Event{
			Type:       agent.EventToolCallResult,
			ToolCallID: "tc-launch",
			ToolName:   "launch_session",
			Content:    `error: unknown agent id "nope"`,
		}
		events <- agent.Event{Type: agent.EventDone, SessionID: "s1"}
		return nil
	})
	seedSession("sess-launch-fail", "Test", testStubAgentType, t.TempDir())

	_, events := postAGUI(t, "sess-launch-fail", "send to nope")
	for _, ev := range events {
		if ev.Type == "CUSTOM" && ev.Raw["name"] == "open_session" {
			t.Fatalf("unexpected open_session event: %+v", ev.Raw)
		}
	}
}

// Ensure bytes helper compiles for concurrent test.
var _ = bytes.NewReader
