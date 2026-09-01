// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIAvailable_knownHarnesses(t *testing.T) {
	cases := []struct {
		harness string
		bin     string
	}{
		{"claude-code", "claude"},
		{"pi", "pi"},
		{"opencode", "opencode"},
		{"antigravity", "agy"},
	}
	for _, tc := range cases {
		_, lookErr := exec.LookPath(tc.bin)
		want := lookErr == nil
		if got := CLIAvailable(tc.harness); got != want {
			t.Fatalf("CLIAvailable(%q) = %v, want %v (LookPath %s: %v)", tc.harness, got, want, tc.bin, lookErr)
		}
	}
}

func TestCLIAvailable_codexUsesNUIPath(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "codex-fake")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NUI_CODEX_PATH", fake)
	if !CLIAvailable("codex") {
		t.Fatal("expected codex available via NUI_CODEX_PATH")
	}
	t.Setenv("NUI_CODEX_PATH", filepath.Join(tmp, "missing"))
	// Fall back to LookPath / known paths — just ensure it does not panic.
	_ = CLIAvailable("codex")
}

func TestCLIAvailable_antigravityUsesNUIPath(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "agy-fake")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NUI_ANTIGRAVITY_PATH", fake)
	if !CLIAvailable("antigravity") {
		t.Fatal("expected antigravity available via NUI_ANTIGRAVITY_PATH")
	}
	t.Setenv("NUI_ANTIGRAVITY_PATH", filepath.Join(tmp, "missing"))
	_ = CLIAvailable("antigravity")
}

func TestCLIAvailable_unknownAlwaysTrue(t *testing.T) {
	if !CLIAvailable("docker") {
		t.Fatal("non-CLI harness types should report available")
	}
	if !CLIAvailable("api") {
		t.Fatal("api should report available at CLI layer")
	}
}
