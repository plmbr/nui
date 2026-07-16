// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "testing"

func TestProgrammaticHarnessExtrasLoopSessionID(t *testing.T) {
	extra := programmaticHarnessExtras(RunRequest{
		LoopSessionID: "sess-loop",
		SessionID:     "harness-resume",
		Message:       "hi",
		WorkingDir:    "/tmp/ws",
	}, "sess-fallback")

	if got := extra["loopSessionId"]; got != "sess-loop" {
		t.Fatalf("loopSessionId=%v want sess-loop", got)
	}
	if got := extra["sessionId"]; got != "harness-resume" {
		t.Fatalf("sessionId=%v want harness-resume", got)
	}
	if got := extra["workingDir"]; got != "/tmp/ws" {
		t.Fatalf("workingDir=%v", got)
	}
}

func TestProgrammaticHarnessExtrasLoopSessionFallback(t *testing.T) {
	extra := programmaticHarnessExtras(RunRequest{Message: "hi"}, "sess-fallback")
	if got := extra["loopSessionId"]; got != "sess-fallback" {
		t.Fatalf("loopSessionId=%v want sess-fallback", got)
	}
	if _, ok := extra["sessionId"]; ok {
		t.Fatalf("unexpected sessionId: %v", extra["sessionId"])
	}
}
