// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"fmt"
	"strings"

	"nui/internal/model"
)

// ValidateOrchestratorRefs checks orchestration member/step registry references.
func ValidateOrchestratorRefs(def model.ADLDefinition) error {
	if err := validateMemberRefs(def); err != nil {
		return err
	}
	return validateStepAgentRefs(def)
}

func validateMemberRefs(def model.ADLDefinition) error {
	members := model.OrchestrationMembers(def)
	if len(members) == 0 {
		return nil
	}
	kind := "orchestration"
	if model.IsCouncilAgent(def) {
		kind = "council"
	} else if model.IsSubAgentsOrchestration(def) {
		kind = "subAgents"
	}
	selfID := model.ADLAgentID(def)
	memberIDs := make([]string, 0, len(members))
	for _, m := range members {
		id := strings.TrimSpace(m.Agent)
		if id == selfID {
			return fmt.Errorf("%s: cannot reference itself (%q)", kind, id)
		}
		member, ok := LookupDefinition(id)
		if !ok {
			return fmt.Errorf("%s: unknown agent %q", kind, id)
		}
		if model.IsOrchestrationAgent(member) {
			return fmt.Errorf("%s: %q has orchestration and cannot be a member", kind, id)
		}
		memberIDs = append(memberIDs, id)
	}
	return detectMemberCycle(memberIDs)
}

func validateStepAgentRefs(def model.ADLDefinition) error {
	for i, step := range model.OrchestrationSteps(def) {
		id := strings.TrimSpace(step.Agent)
		if id == "" {
			continue
		}
		selfID := model.ADLAgentID(def)
		if id == selfID {
			return fmt.Errorf("orchestration.steps[%d]: agent cannot reference the parent (%q)", i, id)
		}
		ref, ok := LookupDefinition(id)
		if !ok {
			return fmt.Errorf("orchestration.steps[%d]: unknown agent %q", i, id)
		}
		if model.IsOrchestrationAgent(ref) {
			return fmt.Errorf("orchestration.steps[%d]: %q has orchestration and cannot be used as a step agent", i, id)
		}
	}
	return nil
}

func detectMemberCycle(memberIDs []string) error {
	visiting := map[string]bool{}
	visited := map[string]bool{}

	var visit func(id string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("orchestration: cycle detected involving agent %q", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		def, ok := LookupDefinition(id)
		if !ok {
			visiting[id] = false
			visited[id] = true
			return nil
		}
		for _, m := range model.OrchestrationMembers(def) {
			subID := strings.TrimSpace(m.Agent)
			if subID == "" {
				continue
			}
			if err := visit(subID); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}

	for _, id := range memberIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
