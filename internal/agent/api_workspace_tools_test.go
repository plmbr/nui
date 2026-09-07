// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nui/internal/hitl"
	"nui/internal/model"
	"nui/internal/skills"
)

func TestProvisionAPIHarnessMaterializesSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	src := t.TempDir()
	skillDir := filepath.Join(src, "demo-skill")
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: demo-skill\ndescription: Demo\n---\n\nBody\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "scripts", "run.sh"), []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatal(err)
	}

	deps := HarnessDeps{
		Skills: []model.ADLSkill{{Name: "demo-skill", Path: skillDir}},
	}
	configDir, err := ProvisionHarnessConfig("api-skills-mat", "api", deps)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(configDir, "skills", "demo-skill")
	if _, err := os.Stat(filepath.Join(dest, "SKILL.md")); err != nil {
		t.Fatalf("SKILL.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "scripts", "run.sh")); err != nil {
		t.Fatalf("scripts/run.sh: %v", err)
	}
}

func TestApproveTool_safeToolsBypassGate(t *testing.T) {
	ag := &APIHarnessAgent{Harness: model.ADLHarness{Type: "api"}}
	req := RunRequest{HarnessPermissions: hitl.PermissionsBypass}
	events := make(chan Event, 4)
	ok, err := ag.approveTool(context.Background(), req, events, "nui-fs__read", nil)
	if err != nil || !ok {
		t.Fatalf("read approve: ok=%v err=%v", ok, err)
	}
	ok, err = ag.approveTool(context.Background(), req, events, "nui-skills__load_skill", nil)
	if err != nil || !ok {
		t.Fatalf("load_skill approve: ok=%v err=%v", ok, err)
	}
}

func TestApproveTool_mutatingRequiresGateEvenWithBypass(t *testing.T) {
	ag := &APIHarnessAgent{Harness: model.ADLHarness{Type: "api"}}
	req := RunRequest{HarnessPermissions: hitl.PermissionsBypass}
	events := make(chan Event, 4)
	for _, tool := range []string{"nui-bash__bash", "nui-fs__write", "nui-agent__save_agent", "nui-agent__update_memory"} {
		ok, err := ag.approveTool(context.Background(), req, events, tool, map[string]any{"command": "echo hi"})
		if err == nil {
			t.Fatalf("%s: expected error without HITL gate", tool)
		}
		if ok {
			t.Fatalf("%s should not auto-approve under bypass", tool)
		}
		if !strings.Contains(err.Error(), "HITL gate") {
			t.Fatalf("%s err = %v", tool, err)
		}
	}
}

func TestApproveTool_policyAllAllowsBash(t *testing.T) {
	ag := &APIHarnessAgent{Harness: model.ADLHarness{Type: "api"}}
	req := RunRequest{
		HarnessPermissions: hitl.PermissionsBypass,
		ToolApprovalPolicy: hitl.ToolApprovalAll,
	}
	events := make(chan Event, 4)
	ok, err := ag.approveTool(context.Background(), req, events, "nui-bash__bash", map[string]any{"command": "echo hi"})
	if err != nil || !ok {
		t.Fatalf("policy all should approve bash: ok=%v err=%v", ok, err)
	}
}

func TestMetadataAppendixInAPIPrompt(t *testing.T) {
	prompt := assembleAPISystemPrompt(HarnessDeps{
		Skills: skills.WithBuiltins(nil),
	})
	if strings.Contains(prompt, "## nui skills") {
		t.Fatal("expected metadata catalog, not full PromptAppendix header")
	}
	if !strings.Contains(prompt, "## Available skills") {
		t.Fatalf("prompt = %q", prompt)
	}
}
