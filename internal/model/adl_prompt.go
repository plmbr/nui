// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import "strings"

// ADLDefaultAutoPrompt is sent when promptMode is auto and no launch or ADL default is set.
const ADLDefaultAutoPrompt = "Follow your system instructions and run."

// ADLPromptModeUser waits for the user to enter a prompt in the UI.
const ADLPromptModeUser = "user"

// ADLPromptModeAuto hides the prompt input and runs with a launch or default prompt.
const ADLPromptModeAuto = "auto"

func orchestrationType(def ADLDefinition) string {
	if def.Orchestration == nil {
		return ""
	}
	return strings.TrimSpace(def.Orchestration.Type)
}

// OrchestrationSteps returns workflow steps when type is workflow.
func OrchestrationSteps(def ADLDefinition) []ADLStep {
	if !IsMultiStepWorkflow(def) || def.Orchestration == nil {
		return nil
	}
	return def.Orchestration.Steps
}

// OrchestrationMembers returns members for subAgents or council.
func OrchestrationMembers(def ADLDefinition) []ADLOrchestrationMember {
	if def.Orchestration == nil {
		return nil
	}
	switch orchestrationType(def) {
	case OrchestrationTypeSubAgents, OrchestrationTypeCouncil:
		return def.Orchestration.Members
	default:
		return nil
	}
}

// IsMultiStepWorkflow reports whether the agent runs a workflow DAG each user turn.
// Multi-step pipelines do not map harness session IDs in agentSessions.
func IsMultiStepWorkflow(def ADLDefinition) bool {
	if orchestrationType(def) != OrchestrationTypeWorkflow || def.Orchestration == nil {
		return false
	}
	steps := def.Orchestration.Steps
	if len(steps) > 1 {
		return true
	}
	for _, step := range steps {
		if strings.TrimSpace(step.Type) == "hitl" {
			return true
		}
	}
	// Single agent step still counts as workflow orchestration mode.
	return len(steps) > 0
}

// IsCouncilAgent reports whether the agent runs multi-member deliberation.
func IsCouncilAgent(def ADLDefinition) bool {
	return orchestrationType(def) == OrchestrationTypeCouncil &&
		def.Orchestration != nil &&
		len(def.Orchestration.Members) > 0
}

// IsSubAgentsOrchestration reports whether the agent runs adaptive chair-loop delegation.
func IsSubAgentsOrchestration(def ADLDefinition) bool {
	return orchestrationType(def) == OrchestrationTypeSubAgents &&
		def.Orchestration != nil &&
		len(def.Orchestration.Members) > 0
}

// HasOrchestration reports whether any multi-agent orchestration mode is active.
func HasOrchestration(def ADLDefinition) bool {
	return IsCouncilAgent(def) || IsSubAgentsOrchestration(def) || IsMultiStepWorkflow(def)
}

// IsOrchestrationAgent reports whether the agent cannot be nested as a member/step agent.
func IsOrchestrationAgent(def ADLDefinition) bool {
	return HasOrchestration(def)
}

// SkipsHarnessSessionPersistence reports whether the top-level agentSessions key is unused.
func SkipsHarnessSessionPersistence(def ADLDefinition) bool {
	return HasOrchestration(def)
}

// IsADLAutoPrompt reports whether the agent runs without waiting for user input.
func IsADLAutoPrompt(def ADLDefinition) bool {
	return strings.TrimSpace(def.PromptMode) == ADLPromptModeAuto
}

// ResolveADLDefaultPrompt returns the ADL defaultPrompt or the built-in auto prompt.
func ResolveADLDefaultPrompt(def ADLDefinition) string {
	if p := strings.TrimSpace(def.DefaultPrompt); p != "" {
		return p
	}
	return ADLDefaultAutoPrompt
}

// ResolveADLLaunchPrompt picks the message for an auto-prompt agent launch.
// override wins when non-empty; otherwise uses ADL defaultPrompt or built-in fallback.
func ResolveADLLaunchPrompt(def ADLDefinition, override string) string {
	if p := strings.TrimSpace(override); p != "" {
		return p
	}
	return ResolveADLDefaultPrompt(def)
}
