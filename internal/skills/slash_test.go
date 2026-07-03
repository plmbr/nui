// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/model"
)

func TestExpandSlashCommand_createAgent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	available := AgentSkills(model.ADLDefinition{})
	got, err := ExpandSlashCommand(Context{}, available, "/create-agent save as my-helper")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(strings.TrimSpace(got), "/create-agent") {
		t.Fatalf("expected expanded skill body, got %q", got)
	}
	if !strings.Contains(got, "Create Agent") {
		t.Fatalf("expected create-agent skill body, got %q", got)
	}
	if !strings.Contains(got, "save as my-helper") {
		t.Fatalf("expected trailing user args, got %q", got)
	}
}

func TestExpandSlashCommand_unknownPassthrough(t *testing.T) {
	msg := "/unknown-command do something"
	got, err := ExpandSlashCommand(Context{}, AgentSkills(model.ADLDefinition{}), msg)
	if err != nil {
		t.Fatal(err)
	}
	if got != msg {
		t.Fatalf("got %q, want passthrough", got)
	}
}

func TestExpandSlashCommand_notSlash(t *testing.T) {
	msg := "please /create-agent later"
	got, err := ExpandSlashCommand(Context{}, AgentSkills(model.ADLDefinition{}), msg)
	if err != nil {
		t.Fatal(err)
	}
	if got != msg {
		t.Fatalf("got %q, want unchanged", got)
	}
}

func TestParseSlashSkillInvocation(t *testing.T) {
	name, args, ok := parseSlashSkillInvocation("  /create-agent extra text  ")
	if !ok || name != "create-agent" || args != "extra text" {
		t.Fatalf("got (%q, %q, %v)", name, args, ok)
	}
}

func TestStripFrontmatter(t *testing.T) {
	in := "---\nname: foo\n---\n\n# Title\n\nBody\n"
	got := stripFrontmatter(in)
	if !strings.Contains(got, "# Title") || strings.Contains(got, "name: foo") {
		t.Fatalf("stripFrontmatter() = %q", got)
	}
}
