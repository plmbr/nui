// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestDrainThroughResultLineConsumesTrailingResult(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"system"}`,
		`{"type":"result","session_id":"sess-1","result":"done"}`,
	}, "\n") + "\n"
	s := &persistentClaudeSession{
		stdout: bufio.NewScanner(strings.NewReader(input)),
	}
	if err := s.drainThroughResultLine(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if s.stdout.Scan() {
		t.Fatalf("unexpected line after drain: %s", s.stdout.Bytes())
	}
}

func TestDrainThroughResultLineNoResult(t *testing.T) {
	s := &persistentClaudeSession{
		stdout: bufio.NewScanner(strings.NewReader(`{"type":"system"}` + "\n")),
	}
	if err := s.drainThroughResultLine(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
}

func TestPersistentClaudeSessionMatchesPrefilledState(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	sess := &persistentClaudeSession{cmd: cmd, binaryPath: "claude"}
	if !sess.matches(&ClaudeCodeAgent{}, RunRequest{}) {
		t.Fatal("expected prefilled session to match default agent request")
	}
}
