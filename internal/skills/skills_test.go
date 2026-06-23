// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"os"
	"path/filepath"
	"testing"

	"loop/internal/model"
	"loop/internal/store"
)

func TestListDiscoversSkillDirectories(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	skillsDir, err := store.SkillsDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"greeting", "greet-with-data"} {
		dir := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list len = %d, want 2: %+v", len(list), list)
	}
}

func TestResolveRefFromFlatSkillDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	skillsDir, err := store.SkillsDir()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(skillsDir, "greeting")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Hi\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve(Context{}, model.ADLSkill{Name: "greeting", Ref: "greeting"})
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("got %q, want %q", got, dir)
	}
}

func TestResolveLocalSkill(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "review-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Review\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "helper.txt"), []byte("extra\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve(Context{}, model.ADLSkill{Name: "review", Path: skillDir})
	if err != nil {
		t.Fatal(err)
	}
	if got != skillDir {
		t.Fatalf("got %q", got)
	}
}

func TestInstallLocalAndResolveRef(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)

	src := filepath.Join(tmp, "code-style")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("# Style\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := InstallLocal("code-style", src); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve(Context{}, model.ADLSkill{Name: "code-style", Ref: "code-style"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(got, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestMaterializeSkillsClaude(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)

	src := filepath.Join(tmp, "greeter")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("# Hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "loop", "sessions", "sess1"), 0700); err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(home, ".loop", "sessions", "sess1")

	skills := []model.ADLSkill{{Name: "greeter", Path: src}}
	if err := MaterializeSkills(Context{}, "claude-code", configDir, skills); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(configDir, "skills", "greeter", "SKILL.md")
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
}

func TestResolveContent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	content := "---\nname: inline\n---\nDo the thing.\n"
	got, err := Resolve(Context{}, model.ADLSkill{
		Name:    "inline",
		Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(got, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Fatalf("content = %q", string(data))
	}
}

func TestCopyDirIncludesExtraFiles(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	if err := os.MkdirAll(filepath.Join(src, "scripts"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("skill\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "scripts", "run.sh"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "scripts", "run.sh")); err != nil {
		t.Fatal(err)
	}
}
