// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"os"
	"testing"

	"nui/internal/agent"
	"nui/internal/store"
)

func TestAppendAndReadRunEvents(t *testing.T) {
	resetRunState()
	dir := t.TempDir()
	store.SetRunsDirOverride(dir)
	t.Cleanup(func() { store.SetRunsDirOverride("") })
	runID := "run-log-test"
	createRunRecord("sess", runID, "hi")

	ev := agent.Event{Type: agent.EventText, Content: "hello"}
	if err := appendRunEvent(runID, 1, ev); err != nil {
		t.Fatal(err)
	}

	entries, err := readRunEvents(runID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Event.Content != "hello" {
		t.Fatalf("entries = %+v", entries)
	}

	entries, err = readRunEvents(runID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries after seq 1, got %+v", entries)
	}
}

func TestPurgeSessionRuns(t *testing.T) {
	resetRunState()
	dir := t.TempDir()
	store.SetRunsDirOverride(dir)
	t.Cleanup(func() { store.SetRunsDirOverride("") })

	createRunRecord("sess-a", "run-a1", "one")
	createRunRecord("sess-a", "run-a2", "two")
	createRunRecord("sess-b", "run-b1", "keep")
	if err := appendRunEvent("run-a1", 1, agent.Event{Type: agent.EventText, Content: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := appendRunEvent("run-a2", 1, agent.Event{Type: agent.EventText, Content: "y"}); err != nil {
		t.Fatal(err)
	}
	if err := appendRunEvent("run-b1", 1, agent.Event{Type: agent.EventText, Content: "z"}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	registerActiveRun("sess-a", "run-a2", cancel)

	purgeSessionRuns("sess-a")

	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected active run cancelled")
	}
	if _, ok := getRunRecord("run-a1"); ok {
		t.Fatal("run-a1 should be purged")
	}
	if _, ok := getRunRecord("run-a2"); ok {
		t.Fatal("run-a2 should be purged")
	}
	if _, ok := getRunRecord("run-b1"); !ok {
		t.Fatal("run-b1 should remain")
	}
	if runs := listSessionRuns("sess-a"); len(runs) != 0 {
		t.Fatalf("sess-a runs = %+v", runs)
	}
	if runs := listSessionRuns("sess-b"); len(runs) != 1 {
		t.Fatalf("sess-b runs = %+v", runs)
	}
	for _, runID := range []string{"run-a1", "run-a2"} {
		path, err := store.RunLogPath(runID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still on disk: %v", runID, err)
		}
	}
	path, err := store.RunLogPath("run-b1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("run-b1 log missing: %v", err)
	}
}
