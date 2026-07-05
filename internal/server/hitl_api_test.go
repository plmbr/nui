// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"loop/internal/hitl"
)

func TestHandleHITLCreateListRespond(t *testing.T) {
	setupTestServerEnv(t)
	seedSession("sess-hitl", "HITL Test", testStubAgentType, t.TempDir())

	createBody := map[string]any{
		"sessionId": "sess-hitl",
		"runId":     "run-1",
		"kind":      hitl.KindQuestion,
		"payload": map[string]any{
			"message": "Pick one",
			"questions": []map[string]any{
				{"id": "q1", "prompt": "Color?", "options": []string{"red", "blue"}},
			},
		},
	}
	raw, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/hitl/requests", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	handleHITLRequests(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body: %s", rec.Code, rec.Body.String())
	}

	var created hitl.Request
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.RequestID == "" {
		t.Fatal("missing requestId")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/hitl/requests?sessionId=sess-hitl&pending=true", nil)
	rec = httptest.NewRecorder()
	handleHITLRequests(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	var pending []hitl.Request
	if err := json.Unmarshal(rec.Body.Bytes(), &pending); err != nil {
		t.Fatal(err)
	}
	if len(pending) == 0 {
		t.Fatal("expected pending requests")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/hitl/requests/"+created.RequestID, nil)
	rec = httptest.NewRecorder()
	handleHITLRequestByID(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}

	respondBody := hitl.RespondInput{
		Status:  hitl.StatusAnswered,
		Answers: map[string]any{"q1": "red"},
		RespondedBy: hitl.RespondedBy{
			Channel: hitl.ChannelLoopUI,
		},
	}
	raw, _ = json.Marshal(respondBody)
	req = httptest.NewRequest(http.MethodPost, "/api/hitl/requests/"+created.RequestID+"/respond", bytes.NewReader(raw))
	rec = httptest.NewRecorder()
	handleHITLRequestByID(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("respond status = %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleHITLCreateDisabledMode(t *testing.T) {
	setupTestServerEnv(t)
	mu.Lock()
	sessions = append(sessions, modelSession("sess-off", "Off", testStubAgentType, t.TempDir()))
	sessions[len(sessions)-1].AgentConfig = map[string]any{
		"hitlMode": hitl.ModeOff,
	}
	mu.Unlock()

	body := map[string]any{
		"sessionId": "sess-off",
		"kind":      hitl.KindQuestion,
		"payload":   map[string]any{"message": "hello"},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/hitl/requests", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	handleHITLRequests(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleHITLCancel(t *testing.T) {
	setupTestServerEnv(t)
	seedSession("sess-cancel", "Cancel", testStubAgentType, t.TempDir())

	createBody := map[string]any{
		"sessionId": "sess-cancel",
		"kind":      hitl.KindApproval,
		"payload":   map[string]any{"message": "approve?"},
	}
	raw, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/hitl/requests", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	handleHITLRequests(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d", rec.Code)
	}
	var created hitl.Request
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/hitl/requests/"+created.RequestID, nil)
	rec = httptest.NewRecorder()
	handleHITLRequestByID(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel status = %d", rec.Code)
	}
}

func TestHandleHITLChannels(t *testing.T) {
	setupTestServerEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/api/hitl-channels", nil)
	rec := httptest.NewRecorder()
	handleHITLChannels(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "loop-ui") {
		t.Fatalf("expected loop-ui channel, body: %s", rec.Body.String())
	}
}

func TestHandleHITLInvalidBody(t *testing.T) {
	setupTestServerEnv(t)
	req := httptest.NewRequest(http.MethodPost, "/api/hitl/requests", strings.NewReader("{"))
	rec := httptest.NewRecorder()
	handleHITLRequests(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
