// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"
)

func TestBuildOpenCodeRunArgs(t *testing.T) {
	args := buildOpenCodeRunArgs(RunRequest{
		Message:    "hello",
		WorkingDir: "/tmp/proj",
		Model:      "anthropic/claude-sonnet-4-6",
	}, "ses_123")
	want := []string{
		"run", "--format", "json",
		"--session", "ses_123",
		"--dir", "/tmp/proj",
		"-m", "anthropic/claude-sonnet-4-6",
		"hello",
	}
	if len(args) != len(want) {
		t.Fatalf("args = %v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q (full: %v)", i, args[i], want[i], args)
		}
	}

	fresh := buildOpenCodeRunArgs(RunRequest{Message: "hi"}, "")
	if fresh[0] != "run" || fresh[1] != "--format" || fresh[2] != "json" {
		t.Fatalf("fresh args = %v", fresh)
	}
	if fresh[len(fresh)-1] != "hi" {
		t.Fatalf("message = %q", fresh[len(fresh)-1])
	}
	for _, arg := range fresh {
		if arg == "--attach" || arg == "--session" {
			t.Fatalf("unexpected resume flag in fresh args: %v", fresh)
		}
	}
}
