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

	path := filepath.Join(home, ".nui", envFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(home, ".nui", legacySecretsFileName)); !os.IsNotExist(err) {
		t.Fatal("legacy secrets.json should be removed after save")
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

func TestMigrateLegacySecretsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".nui")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(dir, legacySecretsFileName)
	body := []byte(`{"env":{"LEGACY_KEY":"from-secrets"}}`)
	if err := os.WriteFile(oldPath, body, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadUserSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Env["LEGACY_KEY"] != "from-secrets" {
		t.Fatalf("env = %+v", loaded.Env)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("legacy secrets.json should be migrated away")
	}
	newPath := filepath.Join(dir, envFileName)
	got, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("env.json = %q want %q", got, body)
	}
}
