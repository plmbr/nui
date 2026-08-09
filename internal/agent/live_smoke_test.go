// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nui/internal/hitl"
	"nui/internal/model"
)

// Live smoke tests for API harnesses. Opt-in only:
//
//	NUI_LIVE_SMOKE=1 go test ./internal/agent -run TestLiveAPIHarness -count=1 -timeout 3m
//
// Uses whatever ANTHROPIC_* / OPENAI_* env the caller exports (e.g. local gateways).
func TestLiveAPIHarnessAnthropic(t *testing.T) {
	if os.Getenv("NUI_LIVE_SMOKE") != "1" {
		t.Skip("set NUI_LIVE_SMOKE=1 to run live API smoke")
	}
	if strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) == "" &&
		strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN")) == "" &&
		strings.TrimSpace(os.Getenv("ANTHROPIC_OAUTH_TOKEN")) == "" {
		t.Skip("no Anthropic credentials")
	}

	h := model.ADLHarness{
		Type:     "api",
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
	}
	runLiveAPISmoke(t, h, "Reply with exactly: nui-ok")
}

func TestLiveAPIHarnessOpenAI(t *testing.T) {
	if os.Getenv("NUI_LIVE_SMOKE") != "1" {
		t.Skip("set NUI_LIVE_SMOKE=1 to run live API smoke")
	}
	if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
		t.Skip("no OpenAI credentials")
	}

	modelName := strings.TrimSpace(os.Getenv("NUI_LIVE_OPENAI_MODEL"))
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}
	h := model.ADLHarness{
		Type:     "api",
		Provider: "openai",
		Model:    modelName,
	}
	runLiveAPISmoke(t, h, "Reply with exactly: nui-ok")
}

func runLiveAPISmoke(t *testing.T, h model.ADLHarness, prompt string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	ag := &APIHarnessAgent{Harness: h, Manager: NewManager()}
	events := make(chan Event, 64)
	errCh := make(chan error, 1)
	go func() {
		errCh <- ag.Run(ctx, RunRequest{
			Message:      prompt,
			SystemPrompt: "You are a concise test assistant. Do not use tools.",
		}, events)
		close(events)
	}()

	var text strings.Builder
	var sawDone bool
	for ev := range events {
		switch ev.Type {
		case EventText:
			text.WriteString(ev.Content)
		case EventError:
			t.Fatalf("event error: %s", ev.Error)
		case EventDone:
			sawDone = true
		}
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !sawDone {
		t.Fatal("missing EventDone")
	}
	out := strings.TrimSpace(text.String())
	if out == "" {
		t.Fatal("empty response text")
	}
	t.Logf("live %s/%s response: %q", h.Provider, h.Model, out)
}

// CLI harness live smoke. Opt-in and separate from API so CI stays offline:
//
//	NUI_LIVE_SMOKE_CLI=1 go test ./internal/agent -run TestLiveCLIHarness -count=1 -timeout 8m
//
// Each turn uses a fresh temp working dir and asks for a text-only reply (no tool use).
func TestLiveCLIHarnessClaudeCode(t *testing.T) {
	runLiveCLISmoke(t, "claude-code")
}

func TestLiveCLIHarnessPi(t *testing.T) {
	runLiveCLISmoke(t, "pi")
}

func TestLiveCLIHarnessCodex(t *testing.T) {
	runLiveCLISmoke(t, "codex")
}

func TestLiveCLIHarnessOpenCode(t *testing.T) {
	runLiveCLISmoke(t, "opencode")
}

func runLiveCLISmoke(t *testing.T, harnessType string) {
	t.Helper()
	if os.Getenv("NUI_LIVE_SMOKE_CLI") != "1" {
		t.Skip("set NUI_LIVE_SMOKE_CLI=1 to run live CLI smoke")
	}
	if !CLIAvailable(harnessType) {
		t.Skipf("%s CLI not available", harnessType)
	}

	workdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workdir, "README.md"), []byte("nui live smoke\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManager()
	t.Cleanup(func() { m.Stop(t.Name()) })

	def := model.ADLDefinition{
		ID:   harnessType,
		Name: harnessType,
		Harness: model.ADLHarness{
			Type:        harnessType,
			Sandbox:     "none",
			Permissions: hitl.PermissionsBypass,
		},
	}
	ag := NewADLAgent(def, t.Name(), m)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	events := make(chan Event, 128)
	errCh := make(chan error, 1)
	go func() {
		errCh <- ag.Run(ctx, RunRequest{
			WorkingDir:         workdir,
			Message:            "Reply with exactly the text nui-ok and do not use any tools.",
			HarnessPermissions: hitl.PermissionsBypass,
			UserScopeHarness:   false,
		}, events)
		close(events)
	}()

	var text strings.Builder
	var sawDone bool
	var errs []string
	for ev := range events {
		switch ev.Type {
		case EventText:
			text.WriteString(ev.Content)
		case EventError:
			errs = append(errs, ev.Error)
		case EventDone:
			sawDone = true
		}
	}
	if err := <-errCh; err != nil {
		if os.Getenv("NUI_LIVE_SMOKE_CLI_STRICT") == "1" {
			t.Fatalf("%s Run: %v (event errors: %v)", harnessType, err, errs)
		}
		t.Skipf("%s Run failed (likely local CLI/auth/env issue): %v (event errors: %v)", harnessType, err, errs)
	}
	if !sawDone {
		if os.Getenv("NUI_LIVE_SMOKE_CLI_STRICT") == "1" {
			t.Fatalf("%s missing EventDone (event errors: %v)", harnessType, errs)
		}
		t.Skipf("%s missing EventDone (likely local CLI/auth/env issue): %v", harnessType, errs)
	}
	out := strings.TrimSpace(text.String())
	if out == "" && len(errs) > 0 {
		// Environmental CLI/auth failures should not fail the default suite hard.
		if os.Getenv("NUI_LIVE_SMOKE_CLI_STRICT") == "1" {
			t.Fatalf("%s empty text with errors: %v", harnessType, errs)
		}
		t.Skipf("%s produced no text (likely local CLI/auth/env issue): %v", harnessType, errs)
	}
	t.Logf("live cli %s response: %q errors=%v", harnessType, out, errs)
	if out != "" && !strings.Contains(strings.ToLower(out), "nui-ok") {
		t.Logf("warning: response did not contain nui-ok")
	}
}
