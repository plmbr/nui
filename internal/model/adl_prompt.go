// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import "strings"

// ADLDefaultAutoPrompt is sent when promptMode is auto and no launch or ADL default is set.
const ADLDefaultAutoPrompt = "Follow your system instructions and run."

// ADLPromptModeUser waits for the user to enter a prompt in the UI.
const ADLPromptModeUser = "user"

// ADLPromptModeAuto hides the prompt input and runs with a launch or default prompt.
const ADLPromptModeAuto = "auto"

// IsMultiStepWorkflow reports whether the agent re-runs all steps on each user turn.
// Multi-step workflows do not map harness session IDs in agentSessions.
func IsMultiStepWorkflow(def ADLDefinition) bool {
	return len(def.Steps) > 0 || def.Kind == "workflow"
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
