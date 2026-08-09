// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunsDirOverride(t *testing.T) {
	dir := t.TempDir()
	SetRunsDirOverride(dir)
	t.Cleanup(func() { SetRunsDirOverride("") })

	got, err := RunsDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("RunsDir() = %q want %q", got, dir)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
}

func TestRunLogPath(t *testing.T) {
	dir := t.TempDir()
	SetRunsDirOverride(dir)
	t.Cleanup(func() { SetRunsDirOverride("") })

	path, err := RunLogPath("run-abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, filepath.Join("run-abc.jsonl")) {
		t.Fatalf("path = %q", path)
	}
}

func TestRemoveRunLog(t *testing.T) {
	dir := t.TempDir()
	SetRunsDirOverride(dir)
	t.Cleanup(func() { SetRunsDirOverride("") })

	path, err := RunLogPath("run-rm")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveRunLog("run-rm"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected log removed, stat err = %v", err)
	}
	if err := RemoveRunLog("run-rm"); err != nil {
		t.Fatalf("removing missing log: %v", err)
	}
}
