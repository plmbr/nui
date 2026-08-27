// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"

	"nui/internal/model"
)

func TestMatchOrchestratorAgent_extensionAgentByPossessive(t *testing.T) {
	candidates := []AgentTypeInfo{
		{ID: "ext:demo-pack/alpha-agent", Label: "alpha_agent / projectx", Available: true},
		{ID: "claude-code", Label: "Claude Code", Available: true},
	}
	agent, score, ok := matchOrchestratorAgent("send a hi to alpha's agent", candidates)
	if !ok {
		t.Fatal("expected match")
	}
	if score < 80 {
		t.Fatalf("score = %d, want >= 80", score)
	}
	if agent.ID != candidates[0].ID {
		t.Fatalf("agent id = %q, want %q", agent.ID, candidates[0].ID)
	}
}

func TestExtractDelegatedPrompt_sendToAgent(t *testing.T) {
	agent := AgentTypeInfo{Label: "alpha_agent / projectx"}
	if got := extractDelegatedPrompt("send a hi to alpha's agent", agent); got != "hi" {
		t.Fatalf("delegated prompt = %q, want hi", got)
	}
}

func TestMatchOrchestratorAgent_ambiguous(t *testing.T) {
	candidates := []AgentTypeInfo{
		{ID: "ext:pack/demo-alpha", Label: "demo_alpha / proj", Available: true},
		{ID: "ext:pack/demo-beta", Label: "demo_beta / proj", Available: true},
	}
	_, _, ok := matchOrchestratorAgent("send a hi to demo's agent", candidates)
	if ok {
		t.Fatal("expected ambiguous match to fail")
	}
	amb := ambiguousOrchestratorCandidates("send a hi to demo's agent", candidates)
	if len(amb) < 2 {
		t.Fatalf("expected ambiguous candidates, got %d", len(amb))
	}
	ids := map[string]bool{}
	for _, c := range amb {
		ids[c.ID] = true
	}
	if !ids["ext:pack/demo-alpha"] || !ids["ext:pack/demo-beta"] {
		t.Fatalf("candidates = %+v", amb)
	}
}

func TestMatchOrchestratorAgent_packageUpdater(t *testing.T) {
	candidates := []AgentTypeInfo{
		{
			ID:            "notebook-suite-updater",
			Label:         "Notebook Suite Updater",
			Description:   "Agent for updating the Notebook Suite package.",
			DefaultPrompt: "Update the Notebook Suite package to the latest version.",
			PromptMode:    model.ADLPromptModeAuto,
			Available:     true,
		},
		{
			ID:          "notebook-suite",
			Label:       "Notebook Suite",
			Description: "Local Notebook Suite agent with notebook tools.",
			Tags:        []string{"notebook", "notebook-suite"},
			Available:   true,
		},
		{
			ID:          "ext:tools/notebook-agent",
			Label:       "Notebook Suite",
			Description: "Notebook agent with suite tools.",
			Tags:        []string{"notebook", "notebook-suite"},
			Available:   true,
		},
	}
	agent, score, ok := matchOrchestratorAgent("update notebook suite", candidates)
	if !ok {
		t.Fatal("expected match")
	}
	if score < 80 {
		t.Fatalf("score = %d, want >= 80", score)
	}
	if agent.ID != "notebook-suite-updater" {
		t.Fatalf("agent id = %q, want notebook-suite-updater", agent.ID)
	}
}

func TestResolveAgentLaunchPrompt_autoAgentUsesDefault(t *testing.T) {
	agent := AgentTypeInfo{
		PromptMode:    model.ADLPromptModeAuto,
		DefaultPrompt: "Update the Notebook Suite package to the latest version.",
	}
	got := resolveAgentLaunchPrompt(agent, explicitDelegatedPrompt("update notebook suite"))
	want := "Update the Notebook Suite package to the latest version."
	if got != want {
		t.Fatalf("prompt = %q, want %q", got, want)
	}
}

func TestMatchOrchestratorAgent_stockAnalyzer(t *testing.T) {
	candidates := []AgentTypeInfo{
		{
			ID:            "acme-stock-analyzer",
			Label:         "Acme Stock Analyzer",
			Description:   "A helpful assistant that can answer questions about Acme stock data.",
			DefaultPrompt: "Search for Acme news in the last 7 days and summarize sentiment.",
			PromptMode:    model.ADLPromptModeAuto,
			Available:     true,
		},
		{
			ID:          "acme-data-analytics",
			Label:       "Acme Data Analytics",
			Description: "Analytics for Acme data",
			Available:   true,
		},
		{
			ID:            "globex-stock-analyzer",
			Label:         "Globex Stock Analyzer",
			Description:   "A helpful assistant that can answer questions about Globex stock data.",
			DefaultPrompt: "Search for Globex news in the last 7 days.",
			PromptMode:    model.ADLPromptModeAuto,
			Available:     true,
		},
	}
	agent, score, ok := matchOrchestratorAgent("analyze acme stock", candidates)
	if !ok {
		t.Fatal("expected match")
	}
	if score < 80 {
		t.Fatalf("score = %d, want >= 80", score)
	}
	if agent.ID != "acme-stock-analyzer" {
		t.Fatalf("agent id = %q, want acme-stock-analyzer", agent.ID)
	}
}

func TestMatchOrchestratorAgent_tfidfParaphrase(t *testing.T) {
	candidates := []AgentTypeInfo{
		{
			ID:          "acme-docs-finder",
			Label:       "Acme Docs Finder",
			Description: "Looks up internal documentation and library references.",
			Tags:        []string{"docs", "lookup"},
			Available:   true,
		},
		{
			ID:          "acme-calendar",
			Label:       "Acme Calendar",
			Description: "Schedules meetings on the calendar.",
			Tags:        []string{"calendar"},
			Available:   true,
		},
	}
	agent, score, ok := matchOrchestratorAgent("find library documentation references", candidates)
	if !ok {
		t.Fatal("expected match")
	}
	if agent.ID != "acme-docs-finder" {
		t.Fatalf("agent id = %q, want acme-docs-finder (score=%d)", agent.ID, score)
	}
}

func TestMatchOrchestratorAgent_prefersBaseOverUnusedVariant(t *testing.T) {
	candidates := []AgentTypeInfo{
		{
			ID:          "acme-search-agent",
			Label:       "Acme Search Agent",
			Description: "Search the Acme knowledge base.",
			Available:   true,
		},
		{
			ID:          "acme-search-agent-short",
			Label:       "Acme Search Agent Short",
			Description: "Search the Acme knowledge base with short answers.",
			Available:   true,
		},
	}
	agent, _, ok := matchOrchestratorAgent("use acme search and lookup widgets", candidates)
	if !ok {
		t.Fatal("expected match")
	}
	if agent.ID != "acme-search-agent" {
		t.Fatalf("agent id = %q, want acme-search-agent", agent.ID)
	}
}

func TestMatchOrchestratorAgent_prefersDescriptionIntentOverShortName(t *testing.T) {
	candidates := []AgentTypeInfo{
		{
			ID:          "ext:tools/acme-task-runtime",
			Label:       "Acme Task Runtime Agent",
			Description: "List, create, manage acme tasks, interact with acme sessions using task MCP",
			Tags:        []string{"managed-agents"},
			Available:   true,
		},
		{
			ID:          "ext:tools/acme",
			Label:       "Acme",
			Description: "General coding agent",
			Tags:        []string{"acme-extension"},
			Available:   true,
		},
	}
	agent, score, ok := matchOrchestratorAgent("list my acme tasks", candidates)
	if !ok {
		t.Fatal("expected match")
	}
	if agent.ID != candidates[0].ID {
		t.Fatalf("agent id = %q, want Acme Task Runtime Agent (score=%d)", agent.ID, score)
	}
}
