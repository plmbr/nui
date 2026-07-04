// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import "fmt"

// ADLHITL configures human-in-the-loop behavior for an agent definition.
type ADLHITL struct {
	Mode     string   `yaml:"mode,omitempty" json:"mode,omitempty"`         // interactive | auto | off
	Required bool     `yaml:"required,omitempty" json:"required,omitempty"`   // semi-autonomous: human involvement mandatory
	Channels []string `yaml:"channels,omitempty" json:"channels,omitempty"`   // loop-ui, ext:...
	TTLSeconds int    `yaml:"ttlSeconds,omitempty" json:"ttlSeconds,omitempty"`
	Approvals  []string `yaml:"approvals,omitempty" json:"approvals,omitempty"` // bash, write, ...
}

// ADLStepHITL configures an orchestration gate step (type: hitl).
type ADLStepHITL struct {
	Kind     string           `yaml:"kind,omitempty" json:"kind,omitempty"` // approval | question | review
	Title    string           `yaml:"title,omitempty" json:"title,omitempty"`
	Message  string           `yaml:"message,omitempty" json:"message,omitempty"`
	Display  []ADLStepDisplay `yaml:"display,omitempty" json:"display,omitempty"`
	Actions  []ADLStepAction  `yaml:"actions,omitempty" json:"actions,omitempty"`
	Questions []map[string]any `yaml:"questions,omitempty" json:"questions,omitempty"`
	Channels []string         `yaml:"channels,omitempty" json:"channels,omitempty"`
}

type ADLStepDisplay struct {
	From string `yaml:"from" json:"from"`
}

type ADLStepAction struct {
	ID    string `yaml:"id" json:"id"`
	Label string `yaml:"label" json:"label"`
}

// ValidateADLHITL returns an error when hitl.required conflicts with promptMode or mode.
func ValidateADLHITL(def ADLDefinition) error {
	if !def.HITL.Required {
		return nil
	}
	if IsADLAutoPrompt(def) {
		return fmt.Errorf("hitl.required cannot be used with promptMode auto")
	}
	mode := def.HITL.Mode
	if mode == "" {
		mode = "interactive"
	}
	if mode != "interactive" {
		return fmt.Errorf("hitl.required requires hitl.mode interactive")
	}
	return nil
}
