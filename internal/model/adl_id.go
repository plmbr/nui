// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

// ADLAgentID returns the canonical agent identifier used by CLI flags and Session.AgentType.
// Falls back to Name when ID is omitted (legacy YAML).
func ADLAgentID(def ADLDefinition) string {
	if def.ID != "" {
		return def.ID
	}
	return def.Name
}

// ADLAgentLabel returns the display name for UI. Falls back to ID when Name is omitted.
func ADLAgentLabel(def ADLDefinition) string {
	if def.Name != "" {
		return def.Name
	}
	return ADLAgentID(def)
}

// NormalizeADLDefinition fills in missing ID or Name from the other field.
func NormalizeADLDefinition(def *ADLDefinition) {
	if def.ID == "" && def.Name != "" {
		def.ID = def.Name
	}
	if def.Name == "" && def.ID != "" {
		def.Name = def.ID
	}
}
