// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"fmt"
	"strings"

	"nui/internal/model"
)

const defaultStepOutputName = "_default"

// stepOutputStore maps step name -> output name -> text.
type stepOutputStore map[string]map[string]string

func newStepOutputStore() stepOutputStore {
	return make(stepOutputStore)
}

func (s stepOutputStore) set(step model.ADLStep, text string) {
	if s[step.Name] == nil {
		s[step.Name] = make(map[string]string)
	}
	if len(step.Outputs) == 0 {
		s[step.Name][defaultStepOutputName] = text
		return
	}
	for _, out := range step.Outputs {
		name := strings.TrimSpace(out.Name)
		if name == "" {
			continue
		}
		s[step.Name][name] = text
	}
}

func (s stepOutputStore) setRaw(stepName, text string) {
	if s[stepName] == nil {
		s[stepName] = make(map[string]string)
	}
	s[stepName][defaultStepOutputName] = text
}

// resolve returns text for a ref like "research" or "research.brief".
func (s stepOutputStore) resolve(ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", false
	}
	parts := strings.SplitN(ref, ".", 2)
	stepName := parts[0]
	outs, ok := s[stepName]
	if !ok || len(outs) == 0 {
		return "", false
	}
	if len(parts) == 1 {
		if text, ok := outs[defaultStepOutputName]; ok {
			return text, true
		}
		for _, text := range outs {
			return text, true
		}
		return "", false
	}
	outputName := parts[1]
	if text, ok := outs[outputName]; ok {
		return text, true
	}
	return "", false
}

func (s stepOutputStore) defaultOutput(stepName string) (string, bool) {
	return s.resolve(stepName)
}

// buildStepMessage constructs the message sent to a step, injecting upstream outputs.
func buildStepMessage(userMsg string, step model.ADLStep, outputs stepOutputStore) string {
	if len(step.Inputs) > 0 {
		var b strings.Builder
		for _, inp := range step.Inputs {
			text, ok := outputs.resolve(inp.From)
			if !ok {
				continue
			}
			label := inp.As
			if label == "" {
				label = inp.From
			}
			fmt.Fprintf(&b, "## %s\n\n%s\n\n", label, text)
		}
		if b.Len() > 0 {
			return b.String() + userMsg
		}
	} else if len(step.DependsOn) > 0 {
		var b strings.Builder
		for _, dep := range step.DependsOn {
			if text, ok := outputs.defaultOutput(dep); ok {
				fmt.Fprintf(&b, "## Output from %s\n\n%s\n\n", dep, text)
			}
		}
		if b.Len() > 0 {
			return b.String() + userMsg
		}
	}
	return userMsg
}
