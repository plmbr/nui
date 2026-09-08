// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nui/internal/appversion"
)

func TestFormatAboutMessage(t *testing.T) {
	got := formatAboutMessage("1.2.3", "4.5.6")
	want := "Self-hosted AI agent sessions\n\nApp version: 1.2.3\nCLI version: 4.5.6"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	got = formatAboutMessage("  ", "")
	if !strings.Contains(got, "App version: dev") {
		t.Fatalf("empty app version should default to dev: %q", got)
	}
	if !strings.Contains(got, "CLI version: unavailable") {
		t.Fatalf("empty CLI version should be unavailable: %q", got)
	}
}

func TestDesktopAboutMessageIncludesVersions(t *testing.T) {
	prev := appversion.Get()
	t.Cleanup(func() { appversion.Set(prev) })
	appversion.Set("1.2.3-test")

	dir := t.TempDir()
	t.Setenv("NUI_INSTALL_DIR", dir)
	script := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 9.9.9-cli; exit 0; fi\nexit 1\n"
	dest := filepath.Join(dir, cliBinaryName())
	if err := os.WriteFile(dest, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	msg := desktopAboutMessage()
	if !strings.Contains(msg, "App version: 1.2.3-test") {
		t.Fatalf("missing app version in about message: %q", msg)
	}
	if !strings.Contains(msg, "CLI version: 9.9.9-cli") {
		t.Fatalf("missing CLI version in about message: %q", msg)
	}
}
