// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import "testing"

func TestValidateADLToolApprovalsAllowlistRequiresTools(t *testing.T) {
	err := ValidateADLToolApprovals(ADLDefinition{
		ToolApprovals: ADLToolApprovals{Policy: ToolApprovalPolicyAllowlist},
	})
	if err == nil {
		t.Fatal("expected error for empty tools with allowlist")
	}
}

func TestValidateADLToolApprovalsDenylistRequiresTools(t *testing.T) {
	err := ValidateADLToolApprovals(ADLDefinition{
		ToolApprovals: ADLToolApprovals{Policy: ToolApprovalPolicyDenylist},
	})
	if err == nil {
		t.Fatal("expected error for empty tools with denylist")
	}
}

func TestValidateADLToolApprovalsValidDenylist(t *testing.T) {
	err := ValidateADLToolApprovals(ADLDefinition{
		ToolApprovals: ADLToolApprovals{
			Policy: ToolApprovalPolicyDenylist,
			Tools:  []string{"Bash", "Write"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateADLToolApprovalsUnknownPolicy(t *testing.T) {
	err := ValidateADLToolApprovals(ADLDefinition{
		ToolApprovals: ADLToolApprovals{Policy: "invalid"},
	})
	if err == nil {
		t.Fatal("expected error for unknown policy")
	}
}
