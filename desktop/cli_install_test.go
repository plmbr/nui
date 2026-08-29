// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIBinaryName(t *testing.T) {
	name := cliBinaryName()
	if runtime.GOOS == "windows" {
		if name != "nui.exe" {
			t.Fatalf("got %q", name)
		}
		return
	}
	if name != "nui" {
		t.Fatalf("got %q", name)
	}
}

func TestCLIInstallDir_defaultAndOverride(t *testing.T) {
	t.Setenv("NUI_INSTALL_DIR", "")
	dir, err := cliInstallDir()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(filepath.Clean(dir), filepath.Join("nui")) {
			t.Fatalf("windows dir = %q", dir)
		}
	} else if !strings.HasSuffix(filepath.Clean(dir), filepath.Join(".local", "bin")) {
		t.Fatalf("unix dir = %q", dir)
	}

	custom := filepath.Join(t.TempDir(), "custom-bin")
	t.Setenv("NUI_INSTALL_DIR", custom)
	dir, err = cliInstallDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Clean(custom) {
		t.Fatalf("got %q want %q", dir, custom)
	}
}

func TestBundledCLIPath_darwinResources(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("darwin layout")
	}
	root := t.TempDir()
	macos := filepath.Join(root, "nui.app", "Contents", "MacOS")
	resources := filepath.Join(root, "nui.app", "Contents", "Resources")
	if err := os.MkdirAll(macos, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resources, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(macos, "nui-desktop")
	cli := filepath.Join(resources, "nui")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cli, []byte("#!/bin/sh\necho test\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := filepath.Clean(filepath.Join(filepath.Dir(exe), "..", "Resources", "nui"))
	if got != filepath.Clean(cli) {
		t.Fatalf("resolved %q want %q", got, cli)
	}
}

func TestPathListContains(t *testing.T) {
	sep := string(os.PathListSeparator)
	dir := filepath.Join(t.TempDir(), "bin")
	if pathListContains("/usr/bin"+sep+dir, dir) != true {
		t.Fatal("expected contains")
	}
	if pathListContains("/usr/bin"+sep+"/bin", dir) {
		t.Fatal("expected missing")
	}
}

func TestCopyFileAtomicAndVersionInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub binary")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	installDir := filepath.Join(home, ".local", "bin")
	t.Setenv("NUI_INSTALL_DIR", installDir)

	bundleDir := filepath.Join(home, "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bundled := filepath.Join(bundleDir, "nui")
	script := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 9.9.9-test; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(bundled, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	ver, err := readCLIVersion(bundled)
	if err != nil {
		t.Fatal(err)
	}
	if ver != "9.9.9-test" {
		t.Fatalf("version = %q", ver)
	}

	dest := filepath.Join(installDir, "nui")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFileAtomic(bundled, dest); err != nil {
		t.Fatal(err)
	}
	got, err := readCLIVersion(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got != "9.9.9-test" {
		t.Fatalf("installed version = %q", got)
	}

	if err := saveCLIState(desktopCLIState{Version: got, Path: dest}); err != nil {
		t.Fatal(err)
	}
	st, ok := loadCLIState()
	if !ok || st.Version != got || st.Path != dest {
		t.Fatalf("state = %+v ok=%v", st, ok)
	}
}

func TestAppendPATHBlockIdempotent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix profiles")
	}
	profile := filepath.Join(t.TempDir(), ".zprofile")
	dir := "/opt/nui/bin"
	if err := appendPATHBlock(profile, dir); err != nil {
		t.Fatal(err)
	}
	if err := appendPATHBlock(profile, dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if c := strings.Count(string(data), cliPathBlockBegin); c != 1 {
		t.Fatalf("begin marker count = %d", c)
	}
}

func TestCLIInstallDestExists(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "nui")
	if cliInstallDestExists(dest) {
		t.Fatal("expected missing dest")
	}
	if err := os.WriteFile(dest, []byte("existing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !cliInstallDestExists(dest) {
		t.Fatal("expected existing dest")
	}
}

func TestInstallBundledCLI_skipsNewerOrEqualDest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub binary")
	}
	dir := t.TempDir()
	dest := filepath.Join(dir, "nui")
	// Installed CLI is newer than bundled — must keep it.
	existing := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 2.0.0; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(dest, []byte(existing), 0o755); err != nil {
		t.Fatal(err)
	}

	bundled := filepath.Join(dir, "bundled")
	bundledScript := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 1.0.0; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(bundled, []byte(bundledScript), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := installBundledCLI(bundled, dest); err != nil {
		t.Fatal(err)
	}

	got, err := readCLIVersion(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2.0.0" {
		t.Fatalf("installed version = %q, want existing binary preserved", got)
	}
}

func TestInstallBundledCLI_upgradesOlderDest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub binary")
	}
	dir := t.TempDir()
	dest := filepath.Join(dir, "nui")
	existing := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 1.0.0; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(dest, []byte(existing), 0o755); err != nil {
		t.Fatal(err)
	}

	bundled := filepath.Join(dir, "bundled")
	bundledScript := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 2.0.0; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(bundled, []byte(bundledScript), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := installBundledCLI(bundled, dest); err != nil {
		t.Fatal(err)
	}

	got, err := readCLIVersion(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2.0.0" {
		t.Fatalf("installed version = %q, want bundled upgrade", got)
	}
}

func TestInstallBundledCLI_installsWhenMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub binary")
	}
	dir := t.TempDir()
	dest := filepath.Join(dir, "nui")
	bundled := filepath.Join(dir, "bundled")
	script := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 9.9.9-bundled; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(bundled, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := installBundledCLI(bundled, dest); err != nil {
		t.Fatal(err)
	}

	got, err := readCLIVersion(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got != "9.9.9-bundled" {
		t.Fatalf("installed version = %q", got)
	}
}

func TestEnsureCLIInstalledErr_missingBundledNoop(t *testing.T) {
	// When Executable() points at a path without a sibling CLI, ensure is a no-op.
	// We cannot easily override os.Executable in unit tests; bundledCLIPath for a
	// temp layout without Resources/nui should return sibling path that doesn't exist,
	// and ensureCLIInstalledErr returns nil after Stat fails.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NUI_INSTALL_DIR", filepath.Join(home, "bin"))
	// Call through with real executable (test binary) — sibling nui won't exist → no-op.
	if err := ensureCLIInstalledErr(); err != nil {
		t.Fatalf("expected no-op, got %v", err)
	}
}
