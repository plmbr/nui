// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtensionEnvRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := SetExtensionEnv("corp-pack", map[string]string{
		"TOKEN":        "abc",
		"NUI_API_URL":  "nope",
		"":             "x",
		"EMPTY":        "  ",
	}); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(home, ".nui", "extension-env.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}

	env := ExtensionEnv("corp-pack")
	if env["TOKEN"] != "abc" {
		t.Fatalf("TOKEN = %q", env["TOKEN"])
	}
	if _, ok := env["NUI_API_URL"]; ok {
		t.Fatal("reserved keys should be dropped")
	}
	keys := ExtensionEnvKeys("corp-pack")
	if len(keys) != 1 || keys[0] != "TOKEN" {
		t.Fatalf("keys = %v", keys)
	}

	if err := SetExtensionEnv("corp-pack", nil); err != nil {
		t.Fatal(err)
	}
	if len(ExtensionEnv("corp-pack")) != 0 {
		t.Fatal("expected empty after clear")
	}
}

func TestRemoveExtensionEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SetExtensionEnv("a", map[string]string{"K": "v"}); err != nil {
		t.Fatal(err)
	}
	if err := SetExtensionEnv("b", map[string]string{"K": "v"}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveExtensionEnv("a"); err != nil {
		t.Fatal(err)
	}
	if len(ExtensionEnv("a")) != 0 {
		t.Fatal("a should be gone")
	}
	if ExtensionEnv("b")["K"] != "v" {
		t.Fatal("b should remain")
	}
}
