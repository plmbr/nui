// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"strings"

	"loop/internal/extensions"
	"loop/internal/model"
	"loop/internal/store"
)

// legacyAgentTypeNames maps old Session.AgentType strings to ADL ids.
var legacyAgentTypeNames = map[string]string{
	"claude-code":     "claude-code",
	"pi":              "pi",
	"codex":           "codex",
	"opencode":        "opencode",
	"docker-claude":   "claude-code",
	"docker-pi":       "pi",
	"docker-opencode": "opencode",
	"Claude Code":     "claude-code",
}

// LookupDefinition resolves an ADL definition by id from builtins, user agents, and extensions.
// It also handles legacy Session.AgentType strings and the "adl:" prefix.
func LookupDefinition(agentType string) (model.ADLDefinition, bool) {
	if mapped, ok := legacyAgentTypeNames[agentType]; ok {
		agentType = mapped
	}
	agentType = strings.TrimPrefix(agentType, "adl:")

	for _, def := range BuiltinAgentDefs() {
		if adlDefMatches(def, agentType) {
			return def, true
		}
	}
	userDefs, _ := store.LoadADLDefinitions()
	for _, def := range userDefs {
		if adlDefMatches(def, agentType) {
			return def, true
		}
	}
	if extensions.Default != nil {
		if def, ok := extensions.Default.FindAgent(agentType); ok {
			return def, true
		}
		if ref, ok := extensions.Default.ResolveHarness(agentType); ok {
			label := ref.Entry.DisplayName
			if label == "" {
				label = ref.Entry.ID
			}
			return model.ADLDefinition{
				ID:          agentType,
				Name:        label,
				Description: ref.Entry.Description,
				Harness:     model.ADLHarness{Type: agentType},
			}, true
		}
	}
	return model.ADLDefinition{}, false
}

func adlDefMatches(def model.ADLDefinition, key string) bool {
	if key == "" {
		return false
	}
	return def.ID == key || def.Name == key || model.ADLAgentID(def) == key
}
