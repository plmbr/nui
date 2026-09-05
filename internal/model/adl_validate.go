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
	if err := validateOrchestration(def); err != nil {
		return err
	}
	return nil
}

func validateOrchestration(def ADLDefinition) error {
	if def.LegacyCouncil != nil {
		return fmt.Errorf("top-level council is no longer supported; use orchestration.type: council")
	}
	if len(def.LegacySteps) > 0 {
		return fmt.Errorf("top-level steps is no longer supported; use orchestration.type: workflow")
	}
	if len(def.LegacySubAgents) > 0 {
		return fmt.Errorf("top-level subAgents is no longer supported; use orchestration.type: subAgents")
	}
	if def.Orchestration == nil {
		return nil
	}
	o := def.Orchestration
	typ := strings.TrimSpace(o.Type)
	switch typ {
	case OrchestrationTypeSubAgents, OrchestrationTypeCouncil, OrchestrationTypeWorkflow:
	case "":
		return fmt.Errorf("orchestration.type is required (subAgents, council, or workflow)")
	default:
		return fmt.Errorf("orchestration.type: must be subAgents, council, or workflow")
	}

	switch typ {
	case OrchestrationTypeSubAgents:
		return validateSubAgentsOrchestration(o)
	case OrchestrationTypeCouncil:
		return validateCouncilOrchestration(o)
	case OrchestrationTypeWorkflow:
		return validateWorkflowOrchestration(o)
	}
	return nil
}

func validateMemberList(members []ADLOrchestrationMember, prefix string) error {
	if len(members) == 0 {
		return fmt.Errorf("%s.members: at least one member is required", prefix)
	}
	seen := map[string]bool{}
	for i, m := range members {
		id := strings.TrimSpace(m.Agent)
		if id == "" {
			return fmt.Errorf("%s.members[%d]: agent is required", prefix, i)
		}
		if seen[id] {
			return fmt.Errorf("%s.members: duplicate agent %q", prefix, id)
		}
		seen[id] = true
	}
	return nil
}

func validateSessionMode(mode, prefix string) error {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return nil
	}
	switch mode {
	case "persistent", "fresh":
		return nil
	default:
		return fmt.Errorf("%s.sessionMode: must be persistent or fresh", prefix)
	}
}

func validateSubAgentsOrchestration(o *ADLOrchestration) error {
	prefix := "orchestration"
	if err := validateMemberList(o.Members, prefix); err != nil {
		return err
	}
	if err := validateSessionMode(o.SessionMode, prefix); err != nil {
		return err
	}
	if o.MaxTurns < 0 {
		return fmt.Errorf("orchestration.maxTurns: must be >= 0")
	}
	if len(o.Steps) > 0 {
		return fmt.Errorf("orchestration: steps are only valid when type is workflow")
	}
	if strings.TrimSpace(o.Rounds) != "" || o.Quorum != 0 || strings.TrimSpace(o.FailurePolicy) != "" ||
		o.MaxParallel != 0 || o.MaxQuestions != 0 {
		return fmt.Errorf("orchestration: rounds/quorum/failurePolicy/maxParallel/maxQuestions are only valid when type is council")
	}
	return nil
}

func validateCouncilOrchestration(o *ADLOrchestration) error {
	prefix := "orchestration"
	if err := validateMemberList(o.Members, prefix); err != nil {
		return err
	}
	if err := validateSessionMode(o.SessionMode, prefix); err != nil {
		return err
	}
	if rounds := strings.TrimSpace(o.Rounds); rounds != "" {
		switch rounds {
		case "independent", "independent+rebuttal", "independent+rebuttal+adjudication":
		default:
			return fmt.Errorf("orchestration.rounds: must be independent, independent+rebuttal, or independent+rebuttal+adjudication")
		}
	}
	if policy := strings.TrimSpace(o.FailurePolicy); policy != "" {
		switch policy {
		case "continue-with-quorum", "fail":
		default:
			return fmt.Errorf("orchestration.failurePolicy: must be continue-with-quorum or fail")
		}
	}
	if o.Quorum < 0 {
		return fmt.Errorf("orchestration.quorum: must be >= 0")
	}
	if o.MaxParallel < 0 {
		return fmt.Errorf("orchestration.maxParallel: must be >= 0")
	}
	if o.MaxQuestions < 0 {
		return fmt.Errorf("orchestration.maxQuestions: must be >= 0")
	}
	if o.MaxTurns != 0 {
		return fmt.Errorf("orchestration: maxTurns is only valid when type is subAgents")
	}
	if len(o.Steps) > 0 {
		return fmt.Errorf("orchestration: steps are only valid when type is workflow")
	}
	return nil
}

func validateWorkflowOrchestration(o *ADLOrchestration) error {
	if len(o.Members) > 0 {
		return fmt.Errorf("orchestration: members are only valid when type is subAgents or council")
	}
	if o.MaxTurns != 0 || strings.TrimSpace(o.Rounds) != "" || o.Quorum != 0 ||
		strings.TrimSpace(o.FailurePolicy) != "" || o.MaxParallel != 0 || o.MaxQuestions != 0 ||
		strings.TrimSpace(o.SessionMode) != "" || strings.TrimSpace(o.MemberTimeout) != "" {
		return fmt.Errorf("orchestration: member/council fields are not valid when type is workflow")
	}
	if len(o.Steps) == 0 {
		return fmt.Errorf("orchestration.steps: at least one step is required when type is workflow")
	}
	stepNames := map[string]bool{}
	for _, step := range o.Steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			return fmt.Errorf("step name is required")
		}
		if stepNames[name] {
			return fmt.Errorf("duplicate step name %q", name)
		}
		stepNames[name] = true
	}
	for _, step := range o.Steps {
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
var CLIHarnessTypes = []string{"claude-code", "pi", "codex", "opencode", "antigravity"}

// IsCLIHarnessType reports whether t is a CLI subprocess harness.
func IsCLIHarnessType(t string) bool {
	switch strings.TrimSpace(t) {
	case "claude-code", "pi", "codex", "opencode", "antigravity":
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
		return fmt.Errorf("allowedHarnesses requires harness.type to be a CLI harness (claude-code, pi, codex, opencode, antigravity); got %q", defaultType)
	}
	seen := map[string]bool{}
	for i, raw := range def.AllowedHarnesses {
		t := strings.TrimSpace(raw)
		if t == "" {
			return fmt.Errorf("allowedHarnesses[%d]: harness type is required", i)
		}
		if !IsCLIHarnessType(t) {
			return fmt.Errorf("allowedHarnesses[%d]: %q is not a CLI harness (only claude-code, pi, codex, opencode, antigravity)", i, t)
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
	case "claude-code", "pi", "codex", "opencode", "antigravity", "docker", "devcontainer", "remote", "api":
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
