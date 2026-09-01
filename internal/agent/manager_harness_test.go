// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"
)

func TestGetBuiltinAgentDifferentHarnessTypes(t *testing.T) {
	m := NewManager()
	projectID := "proj-multi-harness"

	claude, err := m.getBuiltinAgent(projectID, "claude-code", nil)
	if err != nil {
		t.Fatal(err)
	}
	if claude.Name() != "claude-code" {
		t.Fatalf("first agent name = %q, want claude-code", claude.Name())
	}

	codex, err := m.getBuiltinAgent(projectID, "codex", nil)
	if err != nil {
		t.Fatal(err)
	}
	if codex.Name() != "codex" {
		t.Fatalf("second agent name = %q, want codex", codex.Name())
	}
	if claude == codex {
		t.Fatal("codex request returned same agent instance as claude-code")
	}
}

func TestGetBuiltinAgentAllCLIHarnessTypes(t *testing.T) {
	m := NewManager()
	projectID := "proj-all-cli"
	for _, typ := range []string{"claude-code", "pi", "codex", "opencode", "antigravity"} {
		ag, err := m.getBuiltinAgent(projectID, typ, nil)
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		if ag.Name() != typ {
			t.Fatalf("%s Name() = %q", typ, ag.Name())
		}
	}
}

func TestGetBuiltinAgentSameHarnessTypeReuses(t *testing.T) {
	m := NewManager()
	projectID := "proj-reuse"

	first, err := m.getBuiltinAgent(projectID, "pi", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.getBuiltinAgent(projectID, "pi", nil)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("expected same pi agent instance on second request")
	}
}

func TestStopBuiltinAgentClearsAllHarnessTypes(t *testing.T) {
	m := NewManager()
	projectID := "proj-stop-all"

	if _, err := m.getBuiltinAgent(projectID, "claude-code", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := m.getBuiltinAgent(projectID, "opencode", nil); err != nil {
		t.Fatal(err)
	}

	m.stopBuiltinAgent(projectID)

	m.builtinMu.Lock()
	defer m.builtinMu.Unlock()
	for key := range m.builtinAgents {
		if projectIDFromCacheKey(key) == projectID {
			t.Fatalf("expected no builtin agents for %q, found key %q", projectID, key)
		}
	}
}

func TestAgentCacheKey(t *testing.T) {
	key := agentCacheKey("sess-1", "codex")
	if projectIDFromCacheKey(key) != "sess-1" {
		t.Fatalf("projectIDFromCacheKey = %q", projectIDFromCacheKey(key))
	}
}
