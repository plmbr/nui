// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import (
	"loop/internal/model"
)

const (
	AgentConfigKeyHitlMode           = "hitlMode"
	AgentConfigKeyHarnessPermissions = "harnessPermissions"
)

// EffectiveMode resolves runtime HITL mode from ADL and session agentConfig.
func EffectiveMode(def model.ADLDefinition, cfg map[string]any) string {
	if v := stringFromConfig(cfg, AgentConfigKeyHitlMode); v != "" {
		return normalizeMode(v)
	}
	if def.HITL.Mode != "" {
		return normalizeMode(def.HITL.Mode)
	}
	if model.IsADLAutoPrompt(def) {
		return ModeAuto
	}
	return ModeInteractive
}

// EffectivePermissions resolves harness permission mode.
func EffectivePermissions(def model.ADLDefinition, cfg map[string]any) string {
	if v := stringFromConfig(cfg, AgentConfigKeyHarnessPermissions); v != "" {
		return normalizePermissions(v)
	}
	if def.Harness.Permissions != "" {
		return normalizePermissions(def.Harness.Permissions)
	}
	return PermissionsBypass
}

// RuntimeAllowed reports whether mid-run MCP HITL is allowed.
func RuntimeAllowed(def model.ADLDefinition, cfg map[string]any) bool {
	return EffectiveMode(def, cfg) == ModeInteractive
}

func normalizeMode(mode string) string {
	switch mode {
	case ModeOff, ModeAuto, ModeInteractive:
		return mode
	default:
		return ModeInteractive
	}
}

func normalizePermissions(p string) string {
	switch p {
	case PermissionsInteractive, PermissionsBypass:
		return p
	default:
		return PermissionsBypass
	}
}

func stringFromConfig(cfg map[string]any, key string) string {
	if cfg == nil {
		return ""
	}
	v, _ := cfg[key].(string)
	return v
}
