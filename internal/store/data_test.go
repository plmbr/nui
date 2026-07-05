// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"testing"

	"loop/internal/model"
)

func TestSaveAndLoadDataRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".loop"), 0o700); err != nil {
		t.Fatal(err)
	}

	initial := Data{
		Sessions: []model.Session{
			{ID: "s1", Name: "One", AgentType: "claude-code", WorkingDir: "/tmp"},
		},
		AgentSessions: map[string]string{"s1": "agent-s1"},
		SessionMessages: map[string][]model.ChatMessage{
			"s1": {
				{ID: "m1", Role: "user", Content: "hello"},
			},
		},
	}
	if err := SaveData(initial); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadData()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Sessions) != 1 || loaded.Sessions[0].ID != "s1" {
		t.Fatalf("sessions = %+v", loaded.Sessions)
	}
	if loaded.AgentSessions["s1"] != "agent-s1" {
		t.Fatalf("agentSessions = %+v", loaded.AgentSessions)
	}
	if len(loaded.SessionMessages["s1"]) != 1 || loaded.SessionMessages["s1"][0].Content != "hello" {
		t.Fatalf("sessionMessages = %+v", loaded.SessionMessages)
	}
}

func TestLoadDataEmptyWhenMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".loop"), 0o700); err != nil {
		t.Fatal(err)
	}

	data, err := LoadData()
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Sessions) != 0 {
		t.Fatalf("sessions = %+v", data.Sessions)
	}
	if data.AgentSessions == nil || data.SessionMessages == nil {
		t.Fatal("expected initialized maps")
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".loop"), 0o700); err != nil {
		t.Fatal(err)
	}

	dark := true
	if err := SaveSettings(Settings{Theme: "dark", SidebarOpen: &dark, DefaultAgentType: "claude-code"}); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Theme != "dark" || loaded.DefaultAgentType != "claude-code" {
		t.Fatalf("settings = %+v", loaded)
	}
}
