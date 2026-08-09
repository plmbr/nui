// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"fmt"
	"strings"
)

// ValidateADLDefinition checks an ADL definition for structural and semantic errors.
func ValidateADLDefinition(def ADLDefinition) error {
	if def.ID == "" && def.Name == "" {
		return fmt.Errorf("agent must define id or name")
	}
	if err := validateHarness(def.Harness, "harness"); err != nil {
		return err
	}
	if err := validateAllowedHarnesses(def); err != nil {
		return err
	}
	if err := ValidateADLSkills(def.AIAssets.Skills); err != nil {
		return fmt.Errorf("aiAssets.skills: %w", err)
	}
	if err := ValidateADLToolApprovals(def); err != nil {
		return err
	}
	if err := ValidateADLHITL(def); err != nil {
		return err
	}
	if err := ValidateADLEvals(def.Evals); err != nil {
		return err
	}
	if err := ValidateADLTags(def.Tags); err != nil {
		return err
	}
	if err := validateSubAgents(def); err != nil {
		return err
	}
	if len(def.Steps) == 0 {
		return nil
	}

	stepNames := map[string]bool{}
	for _, step := range def.Steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			return fmt.Errorf("step name is required")
		}
		if stepNames[name] {
			return fmt.Errorf("duplicate step name %q", name)
		}
		stepNames[name] = true
	}

	for _, step := range def.Steps {
		if err := validateStep(step, stepNames); err != nil {
			return err
		}
	}
	return nil
}

func validateStep(step ADLStep, stepNames map[string]bool) error {
	name := strings.TrimSpace(step.Name)
	stepType := strings.TrimSpace(step.Type)
	if stepType == "" {
		stepType = "agent"
	}
	switch stepType {
	case "agent":
		harness := step.Harness
		if harness != nil {
			if err := validateHarness(*harness, fmt.Sprintf("step %q harness", name)); err != nil {
				return err
			}
		}
		if err := ValidateADLSkills(step.AIAssets.Skills); err != nil {
			return fmt.Errorf("step %q aiAssets.skills: %w", name, err)
		}
	case "hitl":
		if step.HITL == nil {
			return fmt.Errorf("step %q: hitl block required for type hitl", name)
		}
	default:
		return fmt.Errorf("step %q: unknown type %q", name, stepType)
	}

	for _, dep := range step.DependsOn {
		if !stepNames[dep] {
			return fmt.Errorf("step %q depends on unknown step %q", name, dep)
		}
	}

	for _, inp := range step.Inputs {
		stepRef, _ := splitStepOutputRef(inp.From)
		if stepRef == "" {
			return fmt.Errorf("step %q: input from is required", name)
		}
		if !stepNames[stepRef] {
			return fmt.Errorf("step %q input from %q references unknown step", name, inp.From)
		}
	}

	for _, out := range step.Outputs {
		if strings.TrimSpace(out.Name) == "" {
			return fmt.Errorf("step %q: output name is required", name)
		}
	}

	return nil
}

func validateHarness(h ADLHarness, path string) error {
	t := strings.TrimSpace(h.Type)
	if t == "" {
		t = "claude-code"
	}
	if !isValidHarnessType(t) {
		return fmt.Errorf("%s.type: unknown harness type %q", path, h.Type)
	}
	switch t {
	case "docker":
		if strings.TrimSpace(h.Image) == "" {
			return fmt.Errorf("%s: docker harness requires image", path)
		}
		if h.ContainerPort == 0 {
			return fmt.Errorf("%s: docker harness requires containerPort", path)
		}
	case "remote":
		if strings.TrimSpace(h.Host) == "" {
			return fmt.Errorf("%s: remote harness requires host", path)
		}
		if h.Port == 0 {
			return fmt.Errorf("%s: remote harness requires port", path)
		}
	case "devcontainer":
		inner := strings.TrimSpace(h.InnerHarness)
		if inner == "" {
			return fmt.Errorf("%s: devcontainer harness requires innerHarness", path)
		}
		if !isDevcontainerInnerHarness(inner) {
			return fmt.Errorf("%s.innerHarness: unknown value %q", path, h.InnerHarness)
		}
	}
	if sb := strings.TrimSpace(h.Sandbox); sb != "" && sb != "none" && sb != "bubblewrap" && sb != "docker" {
		return fmt.Errorf("%s.sandbox: unknown value %q", path, sb)
	}
	if p := strings.TrimSpace(h.Permissions); p != "" && p != "interactive" && p != "bypass" {
		return fmt.Errorf("%s.permissions: unknown value %q", path, p)
	}
	return nil
}

// CLIHarnessTypes are harness types allowed in allowedHarnesses (v1).
var CLIHarnessTypes = []string{"claude-code", "pi", "codex", "opencode"}

// IsCLIHarnessType reports whether t is a CLI subprocess harness.
func IsCLIHarnessType(t string) bool {
	switch strings.TrimSpace(t) {
	case "claude-code", "pi", "codex", "opencode":
		return true
	default:
		return false
	}
}

// NormalizeAllowedHarnesses returns the effective CLI harness allowlist for session override.
//
//   - Omitted/empty and harness.type is a CLI harness → all CLIHarnessTypes (default first).
//   - Omitted/empty and harness.type is non-CLI → nil (no CLI session override).
//   - Explicit list → cleaned whitelist that always includes the default harness type.
func NormalizeAllowedHarnesses(def ADLDefinition) []string {
	defaultType := strings.TrimSpace(def.Harness.Type)
	if defaultType == "" {
		defaultType = "claude-code"
	}
	seen := map[string]bool{}
	var out []string
	add := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		out = append(out, t)
	}

	if len(def.AllowedHarnesses) == 0 {
		if !IsCLIHarnessType(defaultType) {
			return nil
		}
		add(defaultType)
		for _, t := range CLIHarnessTypes {
			add(t)
		}
		return out
	}

	add(defaultType)
	for _, t := range def.AllowedHarnesses {
		add(t)
	}
	return out
}

// HarnessOverrideAllowed reports whether override may be used for this def.
// Empty override is always allowed (means use default).
func HarnessOverrideAllowed(def ADLDefinition, override string) bool {
	override = strings.TrimSpace(override)
	if override == "" {
		return true
	}
	allowed := NormalizeAllowedHarnesses(def)
	if len(allowed) == 0 {
		return false
	}
	for _, t := range allowed {
		if t == override {
			return true
		}
	}
	return false
}

func validateAllowedHarnesses(def ADLDefinition) error {
	if len(def.AllowedHarnesses) == 0 {
		return nil
	}
	defaultType := strings.TrimSpace(def.Harness.Type)
	if defaultType == "" {
		defaultType = "claude-code"
	}
	if !IsCLIHarnessType(defaultType) {
		return fmt.Errorf("allowedHarnesses requires harness.type to be a CLI harness (claude-code, pi, codex, opencode); got %q", defaultType)
	}
	seen := map[string]bool{}
	for i, raw := range def.AllowedHarnesses {
		t := strings.TrimSpace(raw)
		if t == "" {
			return fmt.Errorf("allowedHarnesses[%d]: harness type is required", i)
		}
		if !IsCLIHarnessType(t) {
			return fmt.Errorf("allowedHarnesses[%d]: %q is not a CLI harness (only claude-code, pi, codex, opencode)", i, t)
		}
		if seen[t] {
			return fmt.Errorf("allowedHarnesses: duplicate %q", t)
		}
		seen[t] = true
	}
	return nil
}

func isValidHarnessType(t string) bool {
	switch t {
	case "claude-code", "pi", "codex", "opencode", "docker", "devcontainer", "remote", "api":
		return true
	}
	if strings.HasPrefix(t, "ext:") {
		rest := strings.TrimPrefix(t, "ext:")
		before, after, ok := strings.Cut(rest, "/")
		return ok && before != "" && after != ""
	}
	// Extension-registered harness agent ids (ext:name/harness-id without ext: prefix in some paths)
	if strings.Contains(t, "/") {
		return true
	}
	return false
}

func isDevcontainerInnerHarness(t string) bool {
	switch t {
	case "claude-code", "pi", "codex", "opencode":
		return true
	default:
		return false
	}
}

// IsDevcontainerInnerHarness reports whether t is a valid innerHarness for devcontainer.
func IsDevcontainerInnerHarness(t string) bool {
	return isDevcontainerInnerHarness(strings.TrimSpace(t))
}

func validateSubAgents(def ADLDefinition) error {
	if len(def.SubAgents) == 0 {
		return nil
	}
	if len(def.Steps) > 0 {
		return fmt.Errorf("subAgents cannot be combined with workflow steps")
	}
	seen := map[string]bool{}
	for i, id := range def.SubAgents {
		id = strings.TrimSpace(id)
		if id == "" {
			return fmt.Errorf("subAgents[%d]: agent id is required", i)
		}
		if seen[id] {
			return fmt.Errorf("subAgents: duplicate agent id %q", id)
		}
		seen[id] = true
	}
	return nil
}

func splitStepOutputRef(ref string) (stepName, outputName string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", ""
	}
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}
