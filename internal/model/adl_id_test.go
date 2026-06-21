// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestADLDefinitionYAML_id(t *testing.T) {
	raw := []byte(`adl: "1.0"
id: data-agent
name: Data Agent
description: Queries analytics data.
harness:
  type: claude-code
`)

	var def ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}
	if def.ID != "data-agent" {
		t.Fatalf("id = %q", def.ID)
	}
	if def.Name != "Data Agent" {
		t.Fatalf("name = %q", def.Name)
	}
	if ADLAgentID(def) != "data-agent" {
		t.Fatalf("ADLAgentID = %q", ADLAgentID(def))
	}
	if ADLAgentLabel(def) != "Data Agent" {
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
