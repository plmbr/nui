// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"fmt"
	"strings"

	"nui/internal/model"
)

// AgentConfigKeyHarnessType is the session agentConfig key for a harness.type override.
const AgentConfigKeyHarnessType = "harnessType"

// AgentConfigKeyHarnessProvider switches the harness to type=api with the given provider.
const AgentConfigKeyHarnessProvider = "harnessProvider"

// HarnessTypeFromConfig reads an optional harness.type override from session agentConfig.
func HarnessTypeFromConfig(cfg map[string]any) string {
	if cfg == nil {
		return ""
	}
	v, _ := cfg[AgentConfigKeyHarnessType].(string)
	return strings.TrimSpace(v)
}

// HarnessProviderFromConfig reads an optional api harness provider override.
func HarnessProviderFromConfig(cfg map[string]any) string {
	if cfg == nil {
		return ""
	}
	v, _ := cfg[AgentConfigKeyHarnessProvider].(string)
	return strings.TrimSpace(v)
}

// SetHarnessTypeOverride stores override in agentConfig (creates map if needed).
func SetHarnessTypeOverride(cfg map[string]any, harnessType string) map[string]any {
	harnessType = strings.TrimSpace(harnessType)
	if harnessType == "" {
		return cfg
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	cfg[AgentConfigKeyHarnessType] = harnessType
	return cfg
}

// ValidateHarnessOverride checks that override is empty or listed in allowedHarnesses.
func ValidateHarnessOverride(def model.ADLDefinition, override string) error {
	override = strings.TrimSpace(override)
	if override == "" {
		return nil
	}
	if !model.IsCLIHarnessType(override) {
		return fmt.Errorf("harness override %q is not a CLI harness", override)
	}
	if !model.HarnessOverrideAllowed(def, override) {
		allowed := model.NormalizeAllowedHarnesses(def)
		if len(allowed) == 0 {
			return fmt.Errorf("agent %q does not allow CLI harness overrides (harness.type is not a CLI harness)", model.ADLAgentID(def))
		}
		return fmt.Errorf("harness %q is not in allowedHarnesses for agent %q (allowed: %s)",
			override, model.ADLAgentID(def), strings.Join(allowed, ", "))
	}
	return nil
}

// ValidateHarnessOverrideAvailable checks allowlist and that the harness CLI is installed.
func ValidateHarnessOverrideAvailable(def model.ADLDefinition, override string) error {
	if err := ValidateHarnessOverride(def, override); err != nil {
		return err
	}
	override = strings.TrimSpace(override)
	if override != "" && !HarnessAvailable(override) {
		return fmt.Errorf("harness %q is not available on this system", override)
	}
	return nil
}

// ApplyHarnessOverride returns a copy of def with harness.type replaced when override is set.
// Other harness fields (model, env, permissions, sandbox) are preserved.
// Per-step harnesses are untouched (they live on Steps and are not copied/mutated here beyond the top-level type).
func ApplyHarnessOverride(def model.ADLDefinition, override string) (model.ADLDefinition, error) {
	override = strings.TrimSpace(override)
	if override == "" {
		return def, nil
	}
	if err := ValidateHarnessOverride(def, override); err != nil {
		return def, err
	}
	def.Harness.Type = override
	return def, nil
}

// ApplySessionHarnessOverride applies agentConfig harness overrides to the definition.
func ApplySessionHarnessOverride(def model.ADLDefinition, agentConfig map[string]any) (model.ADLDefinition, error) {
	if provider := HarnessProviderFromConfig(agentConfig); provider != "" {
		def.Harness.Type = "api"
		def.Harness.Provider = provider
		return def, nil
	}
	return ApplyHarnessOverride(def, HarnessTypeFromConfig(agentConfig))
}
