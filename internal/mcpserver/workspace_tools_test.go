// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpserver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillRootPathAndList(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: demo\ndescription: Hello\n---\n\nBody\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envNuiSkillsRoot, root)

	entries, err := listMaterializedSkills()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "demo" || entries[0].Description != "Hello" {
		t.Fatalf("entries = %+v", entries)
	}
	got, err := skillRootPath("demo")
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("empty skill root")
	}
	if _, err := skillRootPath("../etc"); err == nil {
		t.Fatal("expected reject path escape")
	}
}

func TestResolveHostPathAndSliceLines(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envNuiWorkingDir, dir)
	abs, err := resolveHostPath("file.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "file.txt")
	if abs != want {
		t.Fatalf("got %q want %q", abs, want)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	tilde, err := resolveHostPath("~/Documents")
	if err != nil {
		t.Fatal(err)
	}
	if tilde != filepath.Join(home, "Documents") {
		t.Fatalf("tilde = %q", tilde)
	}
	got := sliceLines("a\nb\nc\nd", 2, 2)
	if got != "b\nc" {
		t.Fatalf("slice = %q", got)
	}
}

func TestFSWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envNuiWorkingDir, dir)
	path := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	abs, err := resolveHostPath("out.txt")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q", data)
	}
}
