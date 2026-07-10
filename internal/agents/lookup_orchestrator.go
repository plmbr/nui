// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"fmt"
	"strings"

	"loop/internal/model"
)

// ValidateOrchestratorRefs checks sub-agent registry references, self-reference, cycles, and workflow targets.
func ValidateOrchestratorRefs(def model.ADLDefinition) error {
	if len(def.SubAgents) == 0 {
		return nil
	}
	selfID := model.ADLAgentID(def)
	for _, id := range def.SubAgents {
		id = strings.TrimSpace(id)
		if id == selfID {
			return fmt.Errorf("subAgents: orchestrator cannot reference itself (%q)", id)
		}
		sub, ok := LookupDefinition(id)
		if !ok {
			return fmt.Errorf("subAgents: unknown agent %q", id)
		}
		if sub.Kind == "workflow" || len(sub.Steps) > 0 {
			return fmt.Errorf("subAgents: %q is a workflow and cannot be a sub-agent", id)
		}
	}
	return detectOrchestratorCycle(def.SubAgents)
}

func detectOrchestratorCycle(subAgentIDs []string) error {
	visiting := map[string]bool{}
	visited := map[string]bool{}

	var visit func(id string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("subAgents: cycle detected involving agent %q", id)
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
		for _, subID := range def.SubAgents {
			subID = strings.TrimSpace(subID)
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

	for _, id := range subAgentIDs {
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
