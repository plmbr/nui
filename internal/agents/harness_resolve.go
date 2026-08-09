// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"fmt"
	"strings"

	"nui/internal/agent"
	"nui/internal/model"
	"nui/internal/store"
)

// CLIHarnessTypes are built-in CLI harness references for defaultHarness settings.
var CLIHarnessTypes = model.CLIHarnessTypes

// HarnessRefForDef returns the settings value for an agent's harness (e.g. api/anthropic, claude-code).
func HarnessRefForDef(def model.ADLDefinition) string {
	if def.Harness.Type == "api" {
		provider := strings.TrimSpace(def.Harness.Provider)
		if provider == "" {
			provider = "anthropic"
		}
		return "api/" + provider
	}
	return strings.TrimSpace(def.Harness.Type)
}

// HarnessFromRef resolves a defaultHarness settings string to an ADLHarness.
func HarnessFromRef(ref string) (model.ADLHarness, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return model.ADLHarness{}, fmt.Errorf("harness ref is required")
	}
	if strings.HasPrefix(ref, "api/") {
		provider := strings.TrimPrefix(ref, "api/")
		for _, def := range BuiltinAgentDefs() {
			if def.Harness.Type != "api" {
				continue
			}
			if strings.TrimSpace(def.Harness.Provider) == provider {
				return def.Harness, nil
			}
		}
		return model.ADLHarness{}, fmt.Errorf("unknown api harness ref %q", ref)
	}
	for _, def := range builtinAgentDefs {
		if def.Harness.Type == ref {
			return def.Harness, nil
		}
	}
	return model.ADLHarness{}, fmt.Errorf("unknown harness ref %q", ref)
}

// HarnessAvailable reports whether the harness referenced by ref can run on this system.
func HarnessAvailable(ref string) bool {
	h, err := HarnessFromRef(ref)
	if err != nil {
		return false
	}
	def := model.ADLDefinition{Harness: h}
	return harnessAvailable(def)
}

func harnessAvailable(def model.ADLDefinition) bool {
	switch def.Harness.Type {
	case "claude-code", "pi", "codex", "opencode":
		return agent.CLIAvailable(def.Harness.Type)
	case "api":
		return agent.APIHarnessAvailable(def.Harness)
	default:
		return true
	}
}

// SelectableHarnessRefs returns available harness refs for the settings UI.
func SelectableHarnessRefs() []struct {
	Ref   string
	Label string
	Group string
} {
	var out []struct {
		Ref   string
		Label string
		Group string
	}
	for _, id := range APIBuiltinOrder {
		for _, def := range BuiltinAgentDefs() {
			if def.ID != id || def.Harness.Type != "api" {
				continue
			}
			ref := HarnessRefForDef(def)
			if !HarnessAvailable(ref) {
				continue
			}
			out = append(out, struct {
				Ref   string
				Label string
				Group string
			}{Ref: ref, Label: def.Name, Group: "API"})
		}
	}
	for _, harnessType := range CLIHarnessTypes {
		ref := harnessType
		if !HarnessAvailable(ref) {
			continue
		}
		label := harnessType
		for _, def := range builtinAgentDefs {
			if def.Harness.Type == harnessType {
				label = def.Name
				break
			}
		}
		out = append(out, struct {
			Ref   string
			Label string
			Group string
		}{Ref: ref, Label: label, Group: "CLI"})
	}
	return out
}

// ResolveDefaultHarness returns the harness for internal agents from settings.
func ResolveDefaultHarness(settings store.Settings) (model.ADLHarness, string, error) {
	ref := strings.TrimSpace(settings.DefaultHarness)
	if ref != "" && HarnessAvailable(ref) {
		h, err := HarnessFromRef(ref)
		return h, ref, err
	}
	ref = ensureDefaultHarness(&settings)
	if ref == "" {
		return model.ADLHarness{}, "", fmt.Errorf("no available harness")
	}
	h, err := HarnessFromRef(ref)
	return h, ref, err
}

// ensureDefaultHarness picks and persists the first available harness when unset.
func ensureDefaultHarness(settings *store.Settings) string {
	if ref := strings.TrimSpace(settings.DefaultHarness); ref != "" && HarnessAvailable(ref) {
		return ref
	}
	for _, id := range APIBuiltinOrder {
		for _, def := range BuiltinAgentDefs() {
			if def.ID != id || def.Harness.Type != "api" {
				continue
			}
			ref := HarnessRefForDef(def)
			if HarnessAvailable(ref) {
				settings.DefaultHarness = ref
				_ = store.SaveSettings(*settings)
				return ref
			}
		}
	}
	for _, harnessType := range CLIHarnessTypes {
		ref := harnessType
		if HarnessAvailable(ref) {
			settings.DefaultHarness = ref
			_ = store.SaveSettings(*settings)
			return ref
		}
	}
	return ""
}

// PickDefaultHarnessRef chooses a harness ref for the UI from settings and availability.
func PickDefaultHarnessRef(settings store.Settings) string {
	ref := strings.TrimSpace(settings.DefaultHarness)
	if ref != "" && HarnessAvailable(ref) {
		return ref
	}
	for _, item := range SelectableHarnessRefs() {
		return item.Ref
	}
	return ""
}
