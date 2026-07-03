// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"testing"

	"loop/internal/model"
)

func TestSchedulesLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	_ = oldHome

	loaded, err := LoadSchedules()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Schedules) != 0 {
		t.Fatalf("expected empty schedules, got %d", len(loaded.Schedules))
	}

	want := SchedulesData{
		Schedules: []model.Schedule{
			{ID: "s1", Name: "Test", AgentType: "auto-agent", Interval: "5m", Enabled: true},
		},
	}
	if err := SaveSchedules(want); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, ".loop", "schedules.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("schedules.json not written: %v", err)
	}

	loaded, err = LoadSchedules()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Schedules) != 1 || loaded.Schedules[0].ID != "s1" {
		t.Fatalf("unexpected loaded schedules: %+v", loaded.Schedules)
	}
}
