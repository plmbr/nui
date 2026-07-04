// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"
)

func TestEventFromHarnessParamsHitlRequest(t *testing.T) {
	ev, ok := eventFromHarnessParams(map[string]any{
		"type":      "hitl_request",
		"requestId": "req-123",
	})
	if !ok {
		t.Fatal("expected hitl_request mapping")
	}
	if ev.Type != EventHITLRequest {
		t.Fatalf("type = %q", ev.Type)
	}
	if ev.Content != "req-123" {
		t.Fatalf("content = %q", ev.Content)
	}

	ev, ok = eventFromHarnessParams(map[string]any{
		"type":       "hitl_request",
		"request_id": "req-456",
	})
	if !ok || ev.Content != "req-456" {
		t.Fatalf("request_id fallback: ok=%v ev=%+v", ok, ev)
	}

	_, ok = eventFromHarnessParams(map[string]any{"type": "hitl_request"})
	if ok {
		t.Fatal("expected missing request id to be ignored")
	}
}

func TestCodexBuildArgsInteractivePermissions(t *testing.T) {
	s := &persistentCodexSession{}
	args := s.buildArgs(RunRequest{
		Message:            "hello",
		HarnessPermissions: "interactive",
	})
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "dangerously-bypass-approvals-and-sandbox") {
		t.Fatalf("interactive mode should not bypass approvals: %v", args)
	}

	args = s.buildArgs(RunRequest{
		Message:            "hello",
		HarnessPermissions: "bypass",
	})
	joined = strings.Join(args, " ")
	if !strings.Contains(joined, "dangerously-bypass-approvals-and-sandbox") {
		t.Fatalf("bypass mode should skip approvals: %v", args)
	}
}
