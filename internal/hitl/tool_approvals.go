// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import (
	"path/filepath"
	"strings"

	"loop/internal/model"
)

const (
	AgentConfigKeyToolApprovalPolicy = "toolApprovalPolicy"
	AgentConfigKeyToolApprovalTools  = "toolApprovalTools"

	ToolApprovalDefault   = model.ToolApprovalPolicyDefault
	ToolApprovalAll       = model.ToolApprovalPolicyAll
	ToolApprovalAllowlist = model.ToolApprovalPolicyAllowlist
	ToolApprovalDenylist  = model.ToolApprovalPolicyDenylist
)

// EffectiveToolApprovals resolves tool approval policy from ADL and session agentConfig.
func EffectiveToolApprovals(def model.ADLDefinition, cfg map[string]any) (policy string, tools []string) {
	if v := stringFromConfig(cfg, AgentConfigKeyToolApprovalPolicy); v != "" {
		policy = normalizeToolApprovalPolicy(v)
		tools = stringSliceFromConfig(cfg, AgentConfigKeyToolApprovalTools)
		return policy, tools
	}
	if p := strings.TrimSpace(def.ToolApprovals.Policy); p != "" {
		policy = normalizeToolApprovalPolicy(p)
		return policy, append([]string(nil), def.ToolApprovals.Tools...)
	}
	if len(def.HITL.Approvals) > 0 {
		return ToolApprovalDenylist, append([]string(nil), def.HITL.Approvals...)
	}
	return ToolApprovalDefault, nil
}

// ShouldAutoApproveTool reports whether a harness tool should skip the Loop approval UI.
func ShouldAutoApproveTool(toolName, policy string, tools []string) bool {
	policy = normalizeToolApprovalPolicy(policy)
	switch policy {
	case ToolApprovalAll:
		return true
	case ToolApprovalAllowlist:
		return toolMatchesAny(toolName, tools)
	case ToolApprovalDenylist:
		return !toolMatchesAny(toolName, tools)
	default:
		return false
	}
}

// ToolsForPermissionsAllow merges ADL auto-approve policy into Claude permissions.allow entries.
func ToolsForPermissionsAllow(policy string, tools, baseAllowed []string) []string {
	policy = normalizeToolApprovalPolicy(policy)
	allowed := append([]string(nil), baseAllowed...)
	switch policy {
	case ToolApprovalAll:
		allowed = appendUniqueTool(allowed, "*")
	case ToolApprovalAllowlist:
		for _, tool := range tools {
			allowed = appendUniqueTool(allowed, tool)
		}
	}
	return allowed
}

func normalizeToolApprovalPolicy(policy string) string {
	switch strings.TrimSpace(policy) {
	case ToolApprovalAll, ToolApprovalAllowlist, ToolApprovalDenylist:
		return strings.TrimSpace(policy)
	default:
		return ToolApprovalDefault
	}
}

func toolMatchesAny(toolName string, patterns []string) bool {
	for _, pattern := range patterns {
		if toolMatchesPattern(toolName, pattern) {
			return true
		}
	}
	return false
}

func toolMatchesPattern(toolName, pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if strings.ContainsAny(pattern, "*?[") {
		ok, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(strings.TrimSpace(toolName)))
		return err == nil && ok
	}
	return strings.EqualFold(strings.TrimSpace(toolName), pattern)
}

func appendUniqueTool(allowed []string, tool string) []string {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return allowed
	}
	for _, existing := range allowed {
		if strings.EqualFold(existing, tool) {
			return allowed
		}
	}
	return append(allowed, tool)
}

func stringSliceFromConfig(cfg map[string]any, key string) []string {
	if cfg == nil {
		return nil
	}
	raw, ok := cfg[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
