// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"fmt"
	"regexp"
	"strings"
)

// ADLEval is a test case for verifying agent behavior.
type ADLEval struct {
	Name        string           `yaml:"name"                  json:"name"`
	Description string           `yaml:"description,omitempty" json:"description,omitempty"`
	Input       string           `yaml:"input,omitempty"       json:"input,omitempty"`
	Messages    []ADLEvalMessage `yaml:"messages,omitempty"    json:"messages,omitempty"`
	Expect      *ADLEvalExpect   `yaml:"expect,omitempty"      json:"expect,omitempty"`
	Tags        []string         `yaml:"tags,omitempty"        json:"tags,omitempty"`
	Timeout     int              `yaml:"timeout,omitempty"     json:"timeout,omitempty"` // seconds
	WorkingDir  string           `yaml:"workingDir,omitempty"  json:"workingDir,omitempty"`
	Disabled    bool             `yaml:"disabled,omitempty"    json:"disabled,omitempty"`
}

// ADLEvalMessage is one turn in a multi-turn eval conversation.
type ADLEvalMessage struct {
	Role    string `yaml:"role"    json:"role"` // user | assistant
	Content string `yaml:"content" json:"content"`
}

// ADLEvalExpect defines how to grade agent output.
type ADLEvalExpect struct {
	Type     string `yaml:"type"               json:"type"` // contains | exact | regex | llm | none
	Value    string `yaml:"value,omitempty"    json:"value,omitempty"`
	Criteria string `yaml:"criteria,omitempty" json:"criteria,omitempty"`
}

// ValidateADLEvals checks eval test cases for structural and semantic errors.
func ValidateADLEvals(evals []ADLEval) error {
	if len(evals) == 0 {
		return nil
	}
	seen := map[string]bool{}
	for i, ev := range evals {
		path := fmt.Sprintf("evals[%d]", i)
		name := strings.TrimSpace(ev.Name)
		if name == "" {
			return fmt.Errorf("%s: name is required", path)
		}
		if seen[name] {
			return fmt.Errorf("%s: duplicate eval name %q", path, name)
		}
		seen[name] = true

		hasInput := strings.TrimSpace(ev.Input) != ""
		hasMessages := len(ev.Messages) > 0
		if hasInput == hasMessages {
			return fmt.Errorf("eval %q: exactly one of input or messages is required", name)
		}
		if hasMessages {
			if err := validateEvalMessages(name, ev.Messages); err != nil {
				return err
			}
		}
		if ev.Expect != nil {
			if err := validateEvalExpect(name, *ev.Expect); err != nil {
				return err
			}
		}
		if ev.Timeout < 0 {
			return fmt.Errorf("eval %q: timeout must be positive", name)
		}
	}
	return nil
}

func validateEvalMessages(evalName string, messages []ADLEvalMessage) error {
	if len(messages) == 0 {
		return fmt.Errorf("eval %q: messages must not be empty", evalName)
	}
	for i, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		if role != "user" && role != "assistant" {
			return fmt.Errorf("eval %q messages[%d]: role must be user or assistant", evalName, i)
		}
		if strings.TrimSpace(msg.Content) == "" {
			return fmt.Errorf("eval %q messages[%d]: content is required", evalName, i)
		}
	}
	lastRole := strings.TrimSpace(messages[len(messages)-1].Role)
	if lastRole != "user" {
		return fmt.Errorf("eval %q: messages must end with a user turn", evalName)
	}
	return nil
}

func validateEvalExpect(evalName string, expect ADLEvalExpect) error {
	t := strings.TrimSpace(expect.Type)
	if t == "" {
		return fmt.Errorf("eval %q expect: type is required", evalName)
	}
	switch t {
	case "contains", "exact", "regex":
		if strings.TrimSpace(expect.Value) == "" {
			return fmt.Errorf("eval %q expect: value is required for type %q", evalName, t)
		}
		if t == "regex" {
			if _, err := regexp.Compile(expect.Value); err != nil {
				return fmt.Errorf("eval %q expect: invalid regex: %w", evalName, err)
			}
		}
	case "llm":
		if strings.TrimSpace(expect.Criteria) == "" {
			return fmt.Errorf("eval %q expect: criteria is required for type llm", evalName)
		}
	case "none":
	default:
		return fmt.Errorf("eval %q expect: unknown type %q", evalName, t)
	}
	return nil
}

// DefaultEvalTimeout returns the default timeout in seconds for running an eval.
func DefaultEvalTimeout(def ADLDefinition) int {
	if strings.TrimSpace(def.Harness.Type) == "devcontainer" {
		return 300
	}
	return 120
}

// EffectiveEvalTimeout returns the timeout for an eval case, using defaults when unset.
func EffectiveEvalTimeout(def ADLDefinition, ev ADLEval) int {
	if ev.Timeout > 0 {
		return ev.Timeout
	}
	return DefaultEvalTimeout(def)
}
