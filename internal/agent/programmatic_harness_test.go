// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "testing"

func TestProgrammaticHarnessExtrasNuiSessionID(t *testing.T) {
	extra := programmaticHarnessExtras(RunRequest{
		NuiSessionID: "sess-nui",
		SessionID:    "harness-resume",
		Message:      "hi",
		WorkingDir:   "/tmp/ws",
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

func TestProgrammaticHarnessExtrasEnv(t *testing.T) {
	extra := programmaticHarnessExtras(RunRequest{
		Message: "hi",
		Env: map[string]string{
			"CUSTOM_ENV_VAR": "https://github.com/org/repo",
		},
	}, "sess-fallback")
	env, ok := extra["env"].(map[string]string)
	if !ok {
		t.Fatalf("env=%T want map[string]string", extra["env"])
	}
	if got := env["CUSTOM_ENV_VAR"]; got != "https://github.com/org/repo" {
		t.Fatalf("CUSTOM_ENV_VAR=%v", got)
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
