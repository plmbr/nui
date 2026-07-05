// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import (
	"testing"

	"loop/internal/model"
)

func TestEffectiveToolApprovalsFromADL(t *testing.T) {
	def := model.ADLDefinition{
		ToolApprovals: model.ADLToolApprovals{
			Policy: ToolApprovalDenylist,
			Tools:  []string{"Bash", "Write"},
		},
	}
	policy, tools := EffectiveToolApprovals(def, nil)
	if policy != ToolApprovalDenylist {
		t.Fatalf("policy = %q", policy)
	}
	if len(tools) != 2 {
		t.Fatalf("tools = %v", tools)
	}
}

func TestEffectiveToolApprovalsSessionOverride(t *testing.T) {
	def := model.ADLDefinition{
		ToolApprovals: model.ADLToolApprovals{
			Policy: ToolApprovalDenylist,
			Tools:  []string{"Bash"},
		},
	}
	cfg := map[string]any{
		AgentConfigKeyToolApprovalPolicy: ToolApprovalAllowlist,
		AgentConfigKeyToolApprovalTools:  []any{"Read", "Grep"},
	}
	policy, tools := EffectiveToolApprovals(def, cfg)
	if policy != ToolApprovalAllowlist {
		t.Fatalf("policy = %q", policy)
	}
	if len(tools) != 2 || tools[0] != "Read" {
		t.Fatalf("tools = %v", tools)
	}
}

func TestEffectiveToolApprovalsLegacyHitlApprovals(t *testing.T) {
	def := model.ADLDefinition{
		HITL: model.ADLHITL{Approvals: []string{"bash", "write"}},
	}
	policy, tools := EffectiveToolApprovals(def, nil)
	if policy != ToolApprovalDenylist {
		t.Fatalf("policy = %q", policy)
	}
	if len(tools) != 2 {
		t.Fatalf("tools = %v", tools)
	}
}

func TestShouldAutoApproveToolPolicies(t *testing.T) {
	tools := []string{"Bash", "Write", "mcp__corp__*"}

	if !ShouldAutoApproveTool("Read", ToolApprovalAll, nil) {
		t.Fatal("all should auto-approve Read")
	}
	if ShouldAutoApproveTool("Bash", ToolApprovalDefault, tools) {
		t.Fatal("default should not auto-approve Bash")
	}
	if !ShouldAutoApproveTool("Read", ToolApprovalAllowlist, []string{"Read", "Grep"}) {
		t.Fatal("allowlist should auto-approve Read")
	}
	if ShouldAutoApproveTool("Bash", ToolApprovalAllowlist, []string{"Read"}) {
		t.Fatal("allowlist should not auto-approve Bash")
	}
	if ShouldAutoApproveTool("Bash", ToolApprovalDenylist, tools) {
		t.Fatal("denylist should not auto-approve Bash")
	}
	if !ShouldAutoApproveTool("Read", ToolApprovalDenylist, tools) {
		t.Fatal("denylist should auto-approve Read")
	}
	if !ShouldAutoApproveTool("mcp__other__deploy", ToolApprovalDenylist, tools) {
		t.Fatal("denylist should auto-approve MCP tools not matching listed patterns")
	}
	if ShouldAutoApproveTool("mcp__corp__deploy", ToolApprovalDenylist, []string{"mcp__corp__*"}) {
		t.Fatal("denylist should prompt for listed MCP glob match")
	}
}

func TestToolsForPermissionsAllow(t *testing.T) {
	base := []string{"mcp__loop-hitl__*"}
	got := ToolsForPermissionsAllow(ToolApprovalAll, nil, base)
	if len(got) != 2 || got[1] != "*" {
		t.Fatalf("all policy = %v", got)
	}
	got = ToolsForPermissionsAllow(ToolApprovalAllowlist, []string{"Read", "Grep"}, base)
	if len(got) != 3 {
		t.Fatalf("allowlist = %v", got)
	}
	got = ToolsForPermissionsAllow(ToolApprovalDenylist, []string{"Bash"}, base)
	if len(got) != 1 {
		t.Fatalf("denylist should not add tools: %v", got)
	}
}
