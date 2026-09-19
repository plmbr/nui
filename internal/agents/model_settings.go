// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"strings"

	"nui/internal/model"
	"nui/internal/store"
)

// BuiltinHarnessSupportsModel reports whether ref identifies a compiled-in
// harness that accepts a model selection.
func BuiltinHarnessSupportsModel(ref string) bool {
	h, err := HarnessFromRef(ref)
	if err != nil {
		return false
	}
	return h.Type == "api" || model.IsCLIHarnessType(h.Type)
}

// HarnessRequiresModel reports whether nui must resolve a model before running
// this harness. Ollama may resolve it by discovering an installed model.
func HarnessRequiresModel(h model.ADLHarness) bool {
	if h.Type == "api" {
		return true
	}
	return h.Type == "antigravity"
}

// ApplyBuiltinHarnessModelSettings overlays the shared model setting for a
// compiled-in agent's harness.
func ApplyBuiltinHarnessModelSettings(def model.ADLDefinition, settings store.Settings) model.ADLDefinition {
	id := model.ADLAgentID(def)
	isBuiltin := false
	for _, builtin := range BuiltinAgentDefs() {
		if model.ADLAgentID(builtin) == id {
			isBuiltin = true
			break
		}
	}
	if !isBuiltin {
		return def
	}
	ref := HarnessRefForDef(def)
	if !BuiltinHarnessSupportsModel(ref) {
		return def
	}
	if selected := strings.TrimSpace(settings.BuiltinHarnessModels[ref]); selected != "" {
		def.Harness.Model = selected
	}
	return def
}

// BuiltinHarnessDefaultModel returns a harness's compiled fallback model.
func BuiltinHarnessDefaultModel(ref string) string {
	h, err := HarnessFromRef(ref)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(h.Model)
}
