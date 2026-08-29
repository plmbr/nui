// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeSettingsUserWins(t *testing.T) {
	sys := Settings{
		Theme:            "light",
		UITheme:          "hawaiian",
		DefaultAgentType: "claude-code",
		DefaultHarness:   "claude-code",
		MemoryUserMode:   "manual",
		MemoryAgentsMode: map[string]string{"a": "manual"},
	}
	user := Settings{
		Theme:            "dark",
		DefaultAgentType: "anthropic",
		MemoryAgentsMode: map[string]string{"a": "auto", "b": "disabled"},
	}
	out := mergeSettings(sys, user)
	if out.Theme != "dark" {
		t.Fatalf("Theme = %q", out.Theme)
	}
	if out.UITheme != "hawaiian" {
		t.Fatalf("UITheme = %q", out.UITheme)
	}
	if out.DefaultAgentType != "anthropic" {
		t.Fatalf("DefaultAgentType = %q", out.DefaultAgentType)
	}
	if out.DefaultHarness != "claude-code" {
		t.Fatalf("DefaultHarness = %q", out.DefaultHarness)
	}
	if out.MemoryAgentsMode["a"] != "auto" || out.MemoryAgentsMode["b"] != "disabled" {
		t.Fatalf("MemoryAgentsMode = %+v", out.MemoryAgentsMode)
	}
}

func TestLoadSettingsMergesSystemAndUser(t *testing.T) {
	home := t.TempDir()
	sys := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(envSystemConfig, sys)
	t.Setenv(envDataDir, "")

	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sys, "settings.json"), []byte(`{
		"theme":"light",
		"defaultAgentType":"claude-code",
		"defaultHarness":"claude-code"
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveSettings(Settings{Theme: "dark", DefaultAgentType: "anthropic"}); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Theme != "dark" {
		t.Fatalf("Theme = %q", loaded.Theme)
	}
	if loaded.DefaultAgentType != "anthropic" {
		t.Fatalf("DefaultAgentType = %q", loaded.DefaultAgentType)
	}
	if loaded.DefaultHarness != "claude-code" {
		t.Fatalf("DefaultHarness = %q", loaded.DefaultHarness)
	}
}

func TestLoadSecretsMergesSystemAndUser(t *testing.T) {
	home := t.TempDir()
	sys := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(envSystemConfig, sys)

	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sys, "secrets.json"), []byte(`{
		"env":{"ORG_KEY":"org","SHARED":"system"}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveSecrets(Secrets{Env: map[string]string{"SHARED": "user", "USER_KEY": "u"}}); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Env["ORG_KEY"] != "org" {
		t.Fatalf("ORG_KEY = %q", loaded.Env["ORG_KEY"])
	}
	if loaded.Env["SHARED"] != "user" {
		t.Fatalf("SHARED = %q", loaded.Env["SHARED"])
	}
	if loaded.Env["USER_KEY"] != "u" {
		t.Fatalf("USER_KEY = %q", loaded.Env["USER_KEY"])
	}
}

func TestUserDirOverride(t *testing.T) {
	custom := t.TempDir()
	t.Setenv(envDataDir, custom)
	dir, err := UserDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != custom {
		t.Fatalf("UserDir = %q, want %q", dir, custom)
	}
}
