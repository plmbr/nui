// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDedupePATH(t *testing.T) {
	sep := string(os.PathListSeparator)
	in := []string{"/a", "/b", "/a", " ", "/c", ""}
	got := joinPATH(dedupePATH(in))
	want := "/a" + sep + "/b" + sep + "/c"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDedupePATH_windowsCase(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only case folding")
	}
	got := dedupePATH([]string{`C:\Bin`, `c:\bin`, `C:\Other`})
	if len(got) != 2 {
		t.Fatalf("got %v, want 2 entries", got)
	}
}

func TestCommonUserPathDirs_includesExisting(t *testing.T) {
	dirs := commonUserPathDirs()
	for _, dir := range dirs {
		st, err := os.Stat(dir)
		if err != nil || !st.IsDir() {
			t.Fatalf("commonUserPathDirs returned missing dir %q: %v", dir, err)
		}
	}
}

func TestEnsureDesktopPATH_findsLocalBin(t *testing.T) {
	home := t.TempDir()
	localBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(localBin, "claude")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	// GUI-like PATH without user bins.
	t.Setenv("PATH", "/usr/bin"+string(os.PathListSeparator)+"/bin")
	t.Setenv("SHELL", "/bin/sh") // avoid slow/noisy login shell in unit tests

	ensureDesktopPATH()

	path := os.Getenv("PATH")
	found := false
	for _, p := range splitPATH(path) {
		if p == localBin {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("PATH missing %s; PATH=%q", localBin, path)
	}
}
