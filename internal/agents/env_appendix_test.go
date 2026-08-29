// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"strings"
	"testing"

	"nui/internal/store"
)

func TestCollectEnvironmentSnapshotUsesDefaultRegistry(t *testing.T) {
	// When Default is unset, snapshot still builds without panicking.
	snap := CollectEnvironmentSnapshot(store.Settings{Theme: "dark"}, 3)
	if snap.Theme != "dark" {
		t.Fatalf("theme = %q", snap.Theme)
	}
	if snap.AgentCount != 3 {
		t.Fatalf("agentCount = %d", snap.AgentCount)
	}
}

func TestFormatEnvironmentAppendix(t *testing.T) {
	out := FormatEnvironmentAppendix(EnvironmentSnapshot{
		Version:          "1.2.3",
		DefaultHarness:   "claude-code",
		Theme:            "dark",
		AgentCount:       5,
		ExtensionCount:   2,
		DisabledExtCount: 1,
		MCPServerCount:   3,
		HarnessCount:     4,
	})
	for _, want := range []string{
		"## Current environment",
		"version: 1.2.3",
		"defaultHarness: claude-code",
		"theme: dark",
		"agents: 5 available",
		"extensions: 2 installed, 1 disabled",
		"mcp servers: 3 configured",
		"harnesses: 4 available",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("appendix missing %q:\n%s", want, out)
		}
	}
}

func TestOrchestratorDefinitionIncludesEnvironmentAppendix(t *testing.T) {
	def := OrchestratorDefinition(store.Settings{Theme: "light", DefaultHarness: "api/anthropic"})
	if !strings.Contains(def.SystemPrompt, "## Current environment") {
		t.Fatalf("expected environment appendix in system prompt")
	}
	if !strings.Contains(def.SystemPrompt, "defaultHarness:") {
		t.Fatalf("expected defaultHarness in appendix")
	}
	if !strings.Contains(def.SystemPrompt, "control_ui") {
		t.Fatalf("expected control_ui in product system prompt")
	}
}

func TestNuiSystemPromptCoversProductModel(t *testing.T) {
	for _, want := range []string{
		"Agents",
		"Harnesses",
		"Extensions",
		"search_agents",
		"launch_session",
		"list_extensions",
		"set_extension_enabled",
	} {
		if !strings.Contains(nuiSystemPrompt, want) {
			t.Fatalf("system prompt missing %q", want)
		}
	}
	if !strings.Contains(LauncherPromptAppendix, "search_agents") {
		t.Fatal("launcher appendix should mention search_agents")
	}
}
