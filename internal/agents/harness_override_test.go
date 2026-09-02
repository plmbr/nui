// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"strings"
	"testing"

	"nui/internal/model"
)

func TestBuiltinCLIAgentsArePinnedToOwnHarness(t *testing.T) {
	for _, def := range builtinAgentDefs {
		allowed := model.NormalizeAllowedHarnesses(def)
		if len(allowed) != 1 || allowed[0] != def.Harness.Type {
			t.Fatalf("%s: AllowedHarnesses effective = %v, want [%s]", def.ID, allowed, def.Harness.Type)
		}
		if err := ValidateHarnessOverride(def, "pi"); err == nil && def.Harness.Type != "pi" {
			t.Fatalf("%s: expected override to pi to be rejected", def.ID)
		}
	}
}

func TestApplyHarnessOverridePinnedRejects(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "pinned",
		Name:    "Pinned",
		Harness: model.ADLHarness{Type: "claude-code", Model: "m1", Sandbox: "none"},
		AllowedHarnesses: []string{"claude-code"},
	}
	_, err := ApplyHarnessOverride(def, "pi")
	if err == nil || !strings.Contains(err.Error(), "not in allowedHarnesses") {
		t.Fatalf("expected allowlist reject, got %v", err)
	}
}

func TestApplyHarnessOverrideOmittedAllowsAnyCLI(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "open",
		Name:    "Open",
		Harness: model.ADLHarness{Type: "claude-code", Model: "m1"},
	}
	got, err := ApplyHarnessOverride(def, "pi")
	if err != nil {
		t.Fatalf("ApplyHarnessOverride: %v", err)
	}
	if got.Harness.Type != "pi" || got.Harness.Model != "m1" {
		t.Fatalf("got %+v", got.Harness)
	}
}

func TestApplyHarnessOverrideAllowlisted(t *testing.T) {
	def := model.ADLDefinition{
		ID:   "portable",
		Name: "Portable",
		Harness: model.ADLHarness{
			Type:    "claude-code",
			Model:   "keep-me",
			Sandbox: "none",
			Env:     map[string]string{"A": "1"},
		},
		AllowedHarnesses: []string{"pi", "codex"},
		Steps: []model.ADLStep{
			{
				Name:    "research",
				Harness: &model.ADLHarness{Type: "opencode", Model: "step-model"},
			},
			{Name: "write"},
		},
	}
	got, err := ApplyHarnessOverride(def, "pi")
	if err != nil {
		t.Fatalf("ApplyHarnessOverride: %v", err)
	}
	if got.Harness.Type != "pi" {
		t.Fatalf("Harness.Type = %q, want pi", got.Harness.Type)
	}
	if got.Harness.Model != "keep-me" || got.Harness.Sandbox != "none" || got.Harness.Env["A"] != "1" {
		t.Fatalf("expected other harness fields preserved, got %+v", got.Harness)
	}
	// Per-step explicit harness must not be rewritten.
	if got.Steps[0].Harness == nil || got.Steps[0].Harness.Type != "opencode" {
		t.Fatalf("step harness mutated: %+v", got.Steps[0].Harness)
	}
	// Steps without harness still inherit top-level (now overridden) at runtime.
	if got.Steps[1].Harness != nil {
		t.Fatalf("unexpected step harness: %+v", got.Steps[1].Harness)
	}
}

func TestValidateHarnessOverrideUnknownCLI(t *testing.T) {
	def := model.ADLDefinition{
		ID:               "portable",
		Harness:          model.ADLHarness{Type: "claude-code"},
		AllowedHarnesses: []string{"pi"},
	}
	err := ValidateHarnessOverride(def, "not-a-harness")
	if err == nil || !strings.Contains(err.Error(), "not a CLI harness") {
		t.Fatalf("expected CLI harness error, got %v", err)
	}
}

func TestHarnessTypeFromConfig(t *testing.T) {
	if got := HarnessTypeFromConfig(nil); got != "" {
		t.Fatalf("got %q", got)
	}
	cfg := SetHarnessTypeOverride(nil, "pi")
	if HarnessTypeFromConfig(cfg) != "pi" {
		t.Fatalf("got %v", cfg)
	}
}

func TestApplySessionHarnessOverrideProvider(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "ide",
		Harness: model.ADLHarness{Type: "claude-code", Model: "m1"},
	}
	got, err := ApplySessionHarnessOverride(def, map[string]any{
		AgentConfigKeyHarnessProvider: "anthropic",
		"model":                       "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Harness.Type != "api" || got.Harness.Provider != "anthropic" {
		t.Fatalf("harness = %+v", got.Harness)
	}
}
