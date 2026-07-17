// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "testing"

func TestProgrammaticHarnessExtrasNuiSessionID(t *testing.T) {
	extra := programmaticHarnessExtras(RunRequest{
		NuiSessionID: "sess-nui",
		SessionID:     "harness-resume",
		Message:       "hi",
		WorkingDir:    "/tmp/ws",
	}, "sess-fallback")

	if got := extra["nuiSessionId"]; got != "sess-nui" {
		t.Fatalf("nuiSessionId=%v want sess-nui", got)
	}
	if got := extra["sessionId"]; got != "harness-resume" {
		t.Fatalf("sessionId=%v want harness-resume", got)
	}
	if got := extra["workingDir"]; got != "/tmp/ws" {
		t.Fatalf("workingDir=%v", got)
	}
}

func TestProgrammaticHarnessExtrasnuiSessionFallback(t *testing.T) {
	extra := programmaticHarnessExtras(RunRequest{Message: "hi"}, "sess-fallback")
	if got := extra["nuiSessionId"]; got != "sess-fallback" {
		t.Fatalf("nuiSessionId=%v want sess-fallback", got)
	}
	if _, ok := extra["sessionId"]; ok {
		t.Fatalf("unexpected sessionId: %v", extra["sessionId"])
	}
}
