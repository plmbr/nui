// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidateADLDefinitionAllowedHarnessesValid(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:   "portable",
		Name: "Portable",
		Harness: ADLHarness{Type: "claude-code"},
		AllowedHarnesses: []string{"pi", "codex"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateADLDefinitionAllowedHarnessesRejectsNonCLI(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:   "bad",
		Name: "Bad",
		Harness: ADLHarness{Type: "claude-code"},
		AllowedHarnesses: []string{"docker"},
	})
	if err == nil || !strings.Contains(err.Error(), "not a CLI harness") {
		t.Fatalf("expected CLI harness error, got %v", err)
	}
}

func TestValidateADLDefinitionAllowedHarnessesRequiresCLIDefault(t *testing.T) {
	err := ValidateADLDefinition(ADLDefinition{
		ID:   "api-agent",
		Name: "API",
		Harness: ADLHarness{Type: "api", Provider: "anthropic"},
		AllowedHarnesses: []string{"claude-code"},
	})
	if err == nil || !strings.Contains(err.Error(), "requires harness.type to be a CLI harness") {
		t.Fatalf("expected CLI default error, got %v", err)
	}
}

func TestNormalizeAllowedHarnessesIncludesDefault(t *testing.T) {
	def := ADLDefinition{
		Harness:          ADLHarness{Type: "claude-code"},
		AllowedHarnesses: []string{"pi", "codex"},
	}
	got := NormalizeAllowedHarnesses(def)
	if len(got) != 3 || got[0] != "claude-code" {
		t.Fatalf("NormalizeAllowedHarnesses = %v, want default first then allowlist", got)
	}
}

func TestNormalizeAllowedHarnessesOmittedMeansAllCLI(t *testing.T) {
	got := NormalizeAllowedHarnesses(ADLDefinition{Harness: ADLHarness{Type: "pi"}})
	if len(got) != len(CLIHarnessTypes) {
		t.Fatalf("NormalizeAllowedHarnesses = %v, want all CLI types", got)
	}
	if got[0] != "pi" {
		t.Fatalf("default should be first, got %v", got)
	}
	for _, want := range CLIHarnessTypes {
		found := false
		for _, t := range got {
			if t == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %q in %v", want, got)
		}
	}
}

func TestNormalizeAllowedHarnessesOmittedNonCLIMeansNone(t *testing.T) {
	got := NormalizeAllowedHarnesses(ADLDefinition{Harness: ADLHarness{Type: "api", Provider: "anthropic"}})
	if got != nil {
		t.Fatalf("expected nil for non-CLI default, got %v", got)
	}
}

func TestHarnessOverrideAllowed(t *testing.T) {
	open := ADLDefinition{Harness: ADLHarness{Type: "claude-code"}}
	if !HarnessOverrideAllowed(open, "") {
		t.Fatal("empty override should be allowed")
	}
	if !HarnessOverrideAllowed(open, "pi") {
		t.Fatal("omitted allowlist should allow any CLI harness")
	}

	pinned := ADLDefinition{
		Harness:          ADLHarness{Type: "claude-code"},
		AllowedHarnesses: []string{"claude-code"},
	}
	if HarnessOverrideAllowed(pinned, "pi") {
		t.Fatal("singleton allowlist should reject other harnesses")
	}
	if !HarnessOverrideAllowed(pinned, "claude-code") {
		t.Fatal("default should be allowed")
	}

	portable := ADLDefinition{
		Harness:          ADLHarness{Type: "claude-code"},
		AllowedHarnesses: []string{"pi"},
	}
	if !HarnessOverrideAllowed(portable, "pi") {
		t.Fatal("allowlisted override should be allowed")
	}
	if !HarnessOverrideAllowed(portable, "claude-code") {
		t.Fatal("default should be allowed via auto-include")
	}
	if HarnessOverrideAllowed(portable, "opencode") {
		t.Fatal("non-listed override should be rejected")
	}

	api := ADLDefinition{Harness: ADLHarness{Type: "api", Provider: "anthropic"}}
	if HarnessOverrideAllowed(api, "pi") {
		t.Fatal("non-CLI default should reject CLI override when allowlist omitted")
	}
}

func TestADLDefinitionYAML_allowedHarnesses(t *testing.T) {
	const raw = `
adl: "1.0"
id: portable
name: Portable
harness:
  type: claude-code
allowedHarnesses:
  - pi
  - codex
`
	var def ADLDefinition
	if err := yaml.Unmarshal([]byte(raw), &def); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(def.AllowedHarnesses) != 2 {
		t.Fatalf("AllowedHarnesses = %v", def.AllowedHarnesses)
	}
}
