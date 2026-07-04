// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
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
