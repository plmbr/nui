// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nui/internal/model"
)

func TestMetadataAppendixUsesDescriptions(t *testing.T) {
	got := MetadataAppendix(Context{}, nil)
	if !strings.Contains(got, "## Available skills") {
		t.Fatalf("missing header: %q", got)
	}
	if !strings.Contains(got, "create-agent") {
		t.Fatal("expected create-agent")
	}
	if !strings.Contains(got, "load_skill") {
		t.Fatal("expected load_skill guidance")
	}
	if strings.Contains(got, "### Skill:") {
		t.Fatal("metadata appendix should not inline full bodies")
	}
}

func TestReadDescriptionBuiltin(t *testing.T) {
	desc, err := ReadDescription(Context{}, model.ADLSkill{Name: "create-agent", Ref: BuiltinRefPrefix + "create-agent"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(desc, "ADL") {
		t.Fatalf("description = %q", desc)
	}
}

func TestReadDescriptionFromFile(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: demo\ndescription: Demo skill for tests\n---\n\n# Demo\n\nFull body here.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	desc, err := ReadDescription(Context{}, model.ADLSkill{Name: "demo", Path: skillDir})
	if err != nil {
		t.Fatal(err)
	}
	if desc != "Demo skill for tests" {
		t.Fatalf("description = %q", desc)
	}
}
