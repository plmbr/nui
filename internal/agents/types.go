// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"loop/internal/agent"
	"loop/internal/extensions"
	"loop/internal/model"
	"loop/internal/store"
)

// TypeInfo describes an agent type for CLI listing.
type TypeInfo struct {
	ID          string
	Label       string
	Description string
	Harness     string
	PromptMode  string
	Source      string // builtin | user | extension
	Available   bool
	File        string // user agents: ~/.loop/agents filename
}

// ListTypes returns builtin, user, and extension agent types without requiring a running server.
func ListTypes() ([]TypeInfo, error) {
	var all []TypeInfo
	for _, def := range builtinAgentDefs {
		all = append(all, typeInfoFromDef(def, "builtin", ""))
	}

	userDefs, err := store.LoadADLDefinitions()
	if err != nil {
		return nil, err
	}
	fileByID, _ := agentFilesByID()
	for _, def := range userDefs {
		if def.Kind == "workflow" {
			continue
		}
		info := typeInfoFromDef(def, "user", "")
		if f, ok := fileByID[model.ADLAgentID(def)]; ok {
			info.File = f
		}
		all = append(all, info)
	}

	reg, err := extensions.LoadRegistry()
	if err == nil && reg != nil {
		for _, def := range reg.AllAgents() {
			if def.Kind == "workflow" {
				continue
			}
			all = append(all, typeInfoFromDef(def, "extension", ""))
		}
		for _, def := range reg.HarnessOnlyAgentTypes() {
			info := typeInfoFromDef(def, "extension", "")
			info.Harness = "extension"
			info.Available = true
			all = append(all, info)
		}
	}

	return all, nil
}

func typeInfoFromDef(def model.ADLDefinition, source, file string) TypeInfo {
	info := TypeInfo{
		ID:          model.ADLAgentID(def),
		Label:       model.ADLAgentLabel(def),
		Description: def.Description,
		Harness:     def.Harness.Type,
		Source:      source,
		File:        file,
		Available:   harnessAvailable(def),
	}
	if model.IsADLAutoPrompt(def) {
		info.PromptMode = model.ADLPromptModeAuto
	}
	return info
}

func harnessAvailable(def model.ADLDefinition) bool {
	switch def.Harness.Type {
	case "claude-code", "pi", "codex", "opencode":
		return agent.CLIAvailable(def.Harness.Type)
	default:
		return true
	}
}
