// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOrchestratorAgentMention(t *testing.T) {
	tests := []struct {
		prompt    string
		mention   string
		delegated string
		ok        bool
	}{
		{"@claude-code fix the bug", "claude-code", "fix the bug", true},
		{"@ext:pack/my-agent summarize logs", "ext:pack/my-agent", "summarize logs", true},
		{"@claude-code", "claude-code", "", true},
		{"@claude-code:[Claude Code] fix the bug", "claude-code", "fix the bug", true},
		{"@ext:pack/suite-updater:[Suite Updater]", "ext:pack/suite-updater", "", true},
		{"  @claude-code  hello  ", "claude-code", "hello", true},
		{"ask @claude-code to help", "", "", false},
		{"no mention here", "", "", false},
		{"@", "", "", false},
	}
	for _, tc := range tests {
		mention, delegated, ok := parseOrchestratorAgentMention(tc.prompt)
		if ok != tc.ok || mention != tc.mention || delegated != tc.delegated {
			t.Fatalf("parseOrchestratorAgentMention(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.prompt, mention, delegated, ok, tc.mention, tc.delegated, tc.ok)
		}
	}
}

func TestTryMentionAgentLaunch(t *testing.T) {
	setupTestServerEnv(t)
	result, ok, err := tryMentionAgentLaunch("@claude-code fix flaky test", t.TempDir())
	if err != nil {
		t.Fatalf("tryMentionAgentLaunch: %v", err)
	}
	if !ok {
		t.Fatal("expected mention launch")
	}
	if result.Session.AgentType != "claude-code" {
		t.Fatalf("agentType = %q", result.Session.AgentType)
	}
	if result.Prompt != "fix flaky test" {
		t.Fatalf("prompt = %q", result.Prompt)
	}
}

func TestTryMentionAgentLaunch_autoPrompt(t *testing.T) {
	home := withTempHome(t)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}
	agentsDir := filepath.Join(home, ".nui", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `adl: "1.0"
id: auto-agent
name: Auto Agent
promptMode: auto
defaultPrompt: Run the default task.
harness:
  type: claude-code
`
	if err := os.WriteFile(filepath.Join(agentsDir, "auto-agent.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	result, ok, err := tryMentionAgentLaunch("@auto-agent", t.TempDir())
	if err != nil {
		t.Fatalf("tryMentionAgentLaunch: %v", err)
	}
	if !ok {
		t.Fatal("expected mention launch")
	}
	if result.Prompt != "Run the default task." {
		t.Fatalf("prompt = %q, want ADL default for auto-mode mention", result.Prompt)
	}
}

func TestTryMentionAgentLaunch_autoAgentUsesDefaultPrompt(t *testing.T) {
	home := withTempHome(t)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}
	agentsDir := filepath.Join(home, ".nui", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `adl: "1.0"
id: auto-agent
name: Auto Agent
promptMode: auto
defaultPrompt: Run the default task.
harness:
  type: claude-code
`
	if err := os.WriteFile(filepath.Join(agentsDir, "auto-agent.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	result, ok, err := tryMentionAgentLaunch("@auto-agent:[Auto Agent]", t.TempDir())
	if err != nil {
		t.Fatalf("tryMentionAgentLaunch: %v", err)
	}
	if !ok {
		t.Fatal("expected mention launch")
	}
	if result.Prompt != "Run the default task." {
		t.Fatalf("prompt = %q", result.Prompt)
	}
}

func TestTryMentionAgentLaunch_autoAgentIgnoresUserText(t *testing.T) {
	home := withTempHome(t)
	resetAllServerState(t)
	if err := initStore(); err != nil {
		t.Fatal(err)
	}
	agentsDir := filepath.Join(home, ".nui", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := `adl: "1.0"
id: auto-agent
name: Auto Agent
promptMode: auto
defaultPrompt: Run the default task.
harness:
  type: claude-code
`
	if err := os.WriteFile(filepath.Join(agentsDir, "auto-agent.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	result, ok, err := tryMentionAgentLaunch("@auto-agent:[Auto Agent] user override", t.TempDir())
	if err != nil {
		t.Fatalf("tryMentionAgentLaunch: %v", err)
	}
	if !ok {
		t.Fatal("expected mention launch")
	}
	if result.Prompt != "Run the default task." {
		t.Fatalf("prompt = %q, want ADL default", result.Prompt)
	}
}

func TestTryMentionAgentLaunch_unknownAgent(t *testing.T) {
	setupTestServerEnv(t)
	_, ok, err := tryMentionAgentLaunch("@not-a-real-agent-id do work", t.TempDir())
	if err != nil {
		t.Fatalf("tryMentionAgentLaunch: %v", err)
	}
	if ok {
		t.Fatal("expected unknown mention to fall through")
	}
}

func TestFindAgentByMentionID(t *testing.T) {
	candidates := []AgentTypeInfo{
		{ID: "claude-code", Label: "Claude Code"},
		{ID: "ext:pack/agent", Label: "Pack Agent"},
	}
	agent, ok := findAgentByMentionID("ext:pack/agent", candidates)
	if !ok || agent.ID != "ext:pack/agent" {
		t.Fatalf("findAgentByMentionID: %+v, %v", agent, ok)
	}
	agent, ok = findAgentByMentionID("ext:pack/agent:[Pack Agent]", candidates)
	if !ok || agent.ID != "ext:pack/agent" {
		t.Fatalf("findAgentByMentionID with display suffix: %+v, %v", agent, ok)
	}
	_, ok = findAgentByMentionID("Claude Code", candidates)
	if ok {
		t.Fatal("expected label lookup to fail")
	}
}

func TestTryMentionAgentLaunch_respectsPromptMode(t *testing.T) {
	setupTestServerEnv(t)
	result, ok, err := tryMentionAgentLaunch("@claude-code", t.TempDir())
	if err != nil {
		t.Fatalf("tryMentionAgentLaunch: %v", err)
	}
	if !ok {
		t.Fatal("expected mention launch")
	}
	if result.Session.AgentType != "claude-code" {
		t.Fatalf("agentType = %q", result.Session.AgentType)
	}
}
