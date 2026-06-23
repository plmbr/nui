// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestADLDefinitionYAML_id(t *testing.T) {
	raw := []byte(`adl: "1.0"
id: test-agent
name: Test Agent
description: Test agent.
harness:
  type: claude-code
`)

	var def ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}
	if def.ID != "test-agent" {
		t.Fatalf("id = %q", def.ID)
	}
	if def.Name != "Test Agent" {
		t.Fatalf("name = %q", def.Name)
	}
	if ADLAgentID(def) != "test-agent" {
		t.Fatalf("ADLAgentID = %q", ADLAgentID(def))
	}
	if ADLAgentLabel(def) != "Test Agent" {
		t.Fatalf("ADLAgentLabel = %q", ADLAgentLabel(def))
	}
}

func TestADLAgentID_legacyNameOnly(t *testing.T) {
	def := ADLDefinition{Name: "legacy-agent"}
	NormalizeADLDefinition(&def)
	if ADLAgentID(def) != "legacy-agent" {
		t.Fatalf("ADLAgentID = %q", ADLAgentID(def))
	}
}
