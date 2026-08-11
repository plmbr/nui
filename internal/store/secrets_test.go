// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := SaveSecrets(Secrets{Env: map[string]string{
		"ANTHROPIC_API_KEY": "sk-test",
		"OPENAI_API_KEY":    "  ",
		"":                  "ignored",
	}}); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(home, ".nui", "secrets.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}

	loaded, err := LoadSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Env["ANTHROPIC_API_KEY"] != "sk-test" {
		t.Fatalf("ANTHROPIC_API_KEY = %q", loaded.Env["ANTHROPIC_API_KEY"])
	}
	if _, ok := loaded.Env["OPENAI_API_KEY"]; ok {
		t.Fatal("empty values should be dropped")
	}
	if SecretEnv("ANTHROPIC_API_KEY") != "sk-test" {
		t.Fatalf("SecretEnv = %q", SecretEnv("ANTHROPIC_API_KEY"))
	}
}

func TestLoadSecretsMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Env) != 0 {
		t.Fatalf("env = %+v", loaded.Env)
	}
}
