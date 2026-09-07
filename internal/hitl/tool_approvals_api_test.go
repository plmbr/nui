// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import "testing"

func TestIsSafeAPIWorkspaceTool(t *testing.T) {
	for _, name := range []string{"read", "nui-fs__read", "mcp__nui-fs__glob", "load_skill", "nui-skills__list_skills"} {
		if !IsSafeAPIWorkspaceTool(name) {
			t.Fatalf("expected safe: %s", name)
		}
	}
	if IsSafeAPIWorkspaceTool("write") {
		t.Fatal("write is not safe")
	}
}

func TestRequiresHostToolApproval(t *testing.T) {
	for _, name := range []string{"bash", "nui-bash__bash", "mcp__nui-bash__bash", "write", "nui-fs__edit"} {
		if !RequiresHostToolApproval(name) {
			t.Fatalf("expected requires approval: %s", name)
		}
	}
	if RequiresHostToolApproval("read") {
		t.Fatal("read should not require host approval")
	}
}
