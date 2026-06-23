// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestADLDefinitionYAML_workingDirInput(t *testing.T) {
	raw := []byte(`adl: "1.0"
name: project-agent
harness:
  type: claude-code
workingDirInput: true
`)

	var def ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}
	if !IsADLWorkingDirInput(def) {
		t.Fatal("expected workingDirInput true")
	}
}

func TestIsADLWorkingDirInput_defaultFalse(t *testing.T) {
	if IsADLWorkingDirInput(ADLDefinition{}) {
		t.Fatal("expected default workingDirInput false")
	}
}
