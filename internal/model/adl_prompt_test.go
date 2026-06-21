// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestResolveADLLaunchPrompt(t *testing.T) {
	def := ADLDefinition{DefaultPrompt: "Custom default."}
	if got := ResolveADLLaunchPrompt(def, "override"); got != "override" {
		t.Fatalf("override = %q", got)
	}
	if got := ResolveADLLaunchPrompt(def, ""); got != "Custom default." {
		t.Fatalf("default = %q", got)
	}
	if got := ResolveADLLaunchPrompt(ADLDefinition{}, ""); got != ADLDefaultAutoPrompt {
		t.Fatalf("builtin = %q", got)
	}
}

func TestADLDefinitionYAML_promptMode(t *testing.T) {
	raw := []byte(`adl: "1.0"
name: auto-agent
harness:
  type: claude-code
promptMode: auto
defaultPrompt: Run the daily check.
`)

	var def ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}
	if !IsADLAutoPrompt(def) {
		t.Fatal("expected auto prompt mode")
	}
	if def.DefaultPrompt != "Run the daily check." {
		t.Fatalf("defaultPrompt = %q", def.DefaultPrompt)
	}
}
