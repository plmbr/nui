// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/model"
)

func TestBuiltinSkillNamesIncludesCreateAgent(t *testing.T) {
	names := BuiltinSkillNames()
	if len(names) == 0 {
		t.Fatal("expected at least one builtin skill")
	}
	found := false
	for _, name := range names {
		if name == "create-agent" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("builtin skills = %v, want create-agent", names)
	}
}

func TestResolveBuiltinCreateAgent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	dir, err := ResolveBuiltin("create-agent")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "name: create-agent") {
		t.Fatalf("SKILL.md = %q", string(data))
	}
}

func TestResolveRefBuiltin(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	got, err := Resolve(Context{}, model.ADLSkill{Name: "create-agent", Ref: BuiltinRefPrefix + "create-agent"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(got, skillFileName)); err != nil {
		t.Fatal(err)
	}
}

func TestWithBuiltinsAppendsMissing(t *testing.T) {
	merged := WithBuiltins([]model.ADLSkill{{Name: "custom-skill", Path: "/tmp/skill"}})
	foundBuiltin := false
	foundCustom := false
	for _, skill := range merged {
		if skill.Name == "create-agent" {
			foundBuiltin = true
			if skill.Ref != BuiltinRefPrefix+"create-agent" {
				t.Fatalf("builtin ref = %q", skill.Ref)
			}
		}
		if skill.Name == "custom-skill" {
			foundCustom = true
		}
	}
	if !foundBuiltin {
		t.Fatal("expected create-agent builtin")
	}
	if !foundCustom {
		t.Fatal("expected custom skill preserved")
	}
}

func TestWithBuiltinsDoesNotDuplicate(t *testing.T) {
	merged := WithBuiltins([]model.ADLSkill{{Name: "create-agent", Path: "/tmp/create-agent"}})
	count := 0
	for _, skill := range merged {
		if skill.Name == "create-agent" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("create-agent count = %d, want 1", count)
	}
}
