// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"fmt"
	"strings"
)

const (
	ToolApprovalPolicyDefault   = "default"
	ToolApprovalPolicyAll       = "all"
	ToolApprovalPolicyAllowlist = "allowlist"
	ToolApprovalPolicyDenylist  = "denylist"
)

// ADLToolApprovals configures selective auto-approve for harness-native tool permissions.
type ADLToolApprovals struct {
	Policy string   `yaml:"policy,omitempty" json:"policy,omitempty"` // default | all | allowlist | denylist
	Tools  []string `yaml:"tools,omitempty"  json:"tools,omitempty"`
}

// ValidateADLToolApprovals returns an error for invalid toolApprovals values.
func ValidateADLToolApprovals(def ADLDefinition) error {
	policy := strings.TrimSpace(def.ToolApprovals.Policy)
	if policy == "" {
		return nil
	}
	switch policy {
	case ToolApprovalPolicyDefault, ToolApprovalPolicyAll, ToolApprovalPolicyAllowlist, ToolApprovalPolicyDenylist:
	default:
		return fmt.Errorf("toolApprovals.policy must be default, all, allowlist, or denylist")
	}
	if policy == ToolApprovalPolicyAllowlist || policy == ToolApprovalPolicyDenylist {
		if len(def.ToolApprovals.Tools) == 0 {
			return fmt.Errorf("toolApprovals.tools is required when policy is %s", policy)
		}
	}
	return nil
}
