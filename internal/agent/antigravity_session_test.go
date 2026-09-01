// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAntigravityCmdEnvAliasesGoogleKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "from-google")
	got := antigravityCmdEnv(nil)
	if got["GEMINI_API_KEY"] != "from-google" {
		t.Fatalf("GEMINI_API_KEY = %q, want from-google", got["GEMINI_API_KEY"])
	}
}

func TestAntigravityCmdEnvPrefersGeminiKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GEMINI_API_KEY", "from-gemini")
	t.Setenv("GOOGLE_API_KEY", "from-google")
	got := antigravityCmdEnv(nil)
	if got["GEMINI_API_KEY"] != "from-gemini" {
		t.Fatalf("GEMINI_API_KEY = %q, want from-gemini", got["GEMINI_API_KEY"])
	}
}

func TestAntigravityCmdEnvUsesSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	dir := filepath.Join(home, ".nui")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "secrets.json")
	if err := os.WriteFile(path, []byte(`{"env":{"GEMINI_API_KEY":"from-secrets"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	got := antigravityCmdEnv(nil)
	if got["GEMINI_API_KEY"] != "from-secrets" {
		t.Fatalf("GEMINI_API_KEY = %q, want from-secrets", got["GEMINI_API_KEY"])
	}
}

func TestIsAntigravityFatalStderr(t *testing.T) {
	if !isAntigravityFatalStderr(`modelProvider is set to "gemini" in settings.json, but the GEMINI_API_KEY environment variable is not set.`) {
		t.Fatal("expected fatal for missing GEMINI_API_KEY")
	}
	if isAntigravityFatalStderr("some transient warning") {
		t.Fatal("expected non-fatal")
	}
}
