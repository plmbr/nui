// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
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
