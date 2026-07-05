// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/agent"
	"loop/internal/model"
)

const testStubAgentType = "claude-code"

func withTempHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(filepath.Join(home, ".loop"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	return home
}

func resetAllServerState(t *testing.T) {
	t.Helper()
	mu.Lock()
	sessions = nil
	sessionMessages = map[string][]model.ChatMessage{}
	agentSessions = map[string]string{}
	mu.Unlock()
	resetRunState()
}

func setupTestServerEnv(t *testing.T) *agent.Manager {
	t.Helper()
	withTempHome(t)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}
	mgr := agent.NewManager()
	extensionManager = mgr
	return mgr
}

func installTestRemoteAgent(t *testing.T, home string) {
	t.Helper()
	agentsDir := filepath.Join(home, ".loop", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `adl: "1.0"
id: test-remote
name: Test Remote
harness:
  type: remote
  host: 127.0.0.1
  port: 9090
`
	if err := os.WriteFile(filepath.Join(agentsDir, "test-remote.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func stubHarnessRun(reply string) agent.HarnessRunHook {
	return func(ctx context.Context, req agent.RunRequest, events chan<- agent.Event) error {
		if reply == "" {
			reply = "stub reply"
		}
		events <- agent.Event{Type: agent.EventText, Content: reply}
		events <- agent.Event{Type: agent.EventDone, SessionID: "stub-session-id"}
		return nil
	}
}

type aguiSSEEvent struct {
	Raw  map[string]any
	Type string
}

func parseSSEEvents(body string) []aguiSSEEvent {
	var out []aguiSSEEvent
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var raw map[string]any
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			continue
		}
		evType, _ := raw["type"].(string)
		out = append(out, aguiSSEEvent{Raw: raw, Type: evType})
	}
	return out
}

func postAGUI(t *testing.T, sessionID, userMessage string) (int, []aguiSSEEvent) {
	t.Helper()
	body := map[string]any{
		"threadId": sessionID,
		"messages": []map[string]string{
			{"id": "user-1", "role": "user", "content": userMessage},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID+"/ag-ui", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	handleSessionAGUI(rec, req, sessionID)
	return rec.Code, parseSSEEvents(rec.Body.String())
}

func postAGUIWithBody(t *testing.T, sessionID string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID+"/ag-ui", body)
	rec := httptest.NewRecorder()
	handleSessionAGUI(rec, req, sessionID)
	return rec
}

func seedSession(id, name, agentType, workingDir string) model.Session {
	s := modelSession(id, name, agentType, workingDir)
	mu.Lock()
	sessions = append(sessions, s)
	mu.Unlock()
	return s
}
