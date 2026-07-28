// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"
)

func TestExtensionDisplayTag(t *testing.T) {
	if got := extensionDisplayTag("Hello World"); got != "hello-world" {
		t.Fatalf("got %q", got)
	}
}

func TestOrchestratorAgentEntries_includesAutoPromptAgents(t *testing.T) {
	all := []AgentTypeInfo{
		{
			ID:            "notebook-suite-updater",
			Label:         "Notebook Suite Updater",
			Description:   "Agent for updating the Notebook Suite package.",
			Available:     true,
			PromptMode:    "auto",
			DefaultPrompt: "Update the Notebook Suite package to the latest version.",
		},
		{ID: "notebook-suite", Label: "Notebook Suite", Available: true},
	}
	entries := orchestratorAgentEntries(all)
	if len(entries) != 2 {
		t.Fatalf("entries = %+v, want both agents listed", entries)
	}
	var updater map[string]any
	for _, entry := range entries {
		if entry["id"] == "notebook-suite-updater" {
			updater = entry
			break
		}
	}
	if updater == nil {
		t.Fatal("expected notebook-suite-updater in list_agents response")
	}
	if updater["launchable"] != true {
		t.Fatalf("launchable = %v, want true for auto prompt agent", updater["launchable"])
	}
	if updater["promptMode"] != "auto" {
		t.Fatalf("promptMode = %v", updater["promptMode"])
	}
}

func TestOrchestratorAgentEntries_includesExtensionAgentByLabel(t *testing.T) {
	all := []AgentTypeInfo{
		{
			ID:          "ext:tools/notebook-agent",
			Label:       "Notebook Suite",
			Description: "Notebook agent with suite tools",
			Available:   true,
			Source:      "extension",
			Tags:        []string{"notebook-suite"},
		},
	}
	entries := orchestratorAgentEntries(all)
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0]["label"] != "Notebook Suite" {
		t.Fatalf("label = %v", entries[0]["label"])
	}
}
