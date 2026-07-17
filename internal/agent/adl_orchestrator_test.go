// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"

	"nui/internal/model"
)

func TestParseRoutingResponse(t *testing.T) {
	candidates := []resolvedSubAgent{
		{id: "hello-world", label: "Hello World", def: model.ADLDefinition{ID: "hello-world", Name: "Hello World"}},
		{id: "code-reviewer", label: "Code Reviewer", def: model.ADLDefinition{ID: "code-reviewer", Name: "Code Reviewer", Description: "Reviews PRs"}},
	}

	picked, ok := parseRoutingResponse("code-reviewer", candidates)
	if !ok || picked.id != "code-reviewer" {
		t.Fatalf("got %+v ok=%v", picked, ok)
	}

	picked, ok = parseRoutingResponse("Hello World\n", candidates)
	if !ok || picked.id != "hello-world" {
		t.Fatalf("label match: got %+v ok=%v", picked, ok)
	}

	_, ok = parseRoutingResponse("", candidates)
	if ok {
		t.Fatal("expected no match for empty")
	}
}

func TestBuildRoutingPromptUsesRegistryDescription(t *testing.T) {
	candidates := []resolvedSubAgent{{
		id:    "code-reviewer",
		label: "Code Reviewer",
		def:   model.ADLDefinition{ID: "code-reviewer", Name: "Code Reviewer", Description: "Reviews code changes"},
	}}
	prompt := buildRoutingPrompt("fix my PR", candidates)
	for _, part := range []string{"code-reviewer", "Code Reviewer", "Reviews code changes", "fix my PR"} {
		if !strings.Contains(prompt, part) {
			t.Fatalf("prompt missing %q: %s", part, prompt)
		}
	}
}
