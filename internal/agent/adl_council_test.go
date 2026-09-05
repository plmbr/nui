// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"
	"time"

	"nui/internal/model"
)

func TestCouncilRoundPlan(t *testing.T) {
	if got := councilRoundPlan("independent"); len(got) != 1 || got[0] != councilRoundPosition {
		t.Fatalf("independent: %v", got)
	}
	if got := councilRoundPlan(""); len(got) != 2 {
		t.Fatalf("default: %v", got)
	}
	if got := councilRoundPlan("independent+rebuttal+adjudication"); len(got) != 3 {
		t.Fatalf("adjudication: %v", got)
	}
}

func TestParseCouncilTimeout(t *testing.T) {
	if d := parseCouncilTimeout("5m"); d != 5*time.Minute {
		t.Fatalf("got %v", d)
	}
	if d := parseCouncilTimeout(""); d != 8*time.Minute {
		t.Fatalf("default %v", d)
	}
}

func TestBuildRebuttalPromptExcludesSelf(t *testing.T) {
	prompt := buildRebuttalPrompt("obj", "a", map[string]string{
		"a": "my position",
		"b": "other position",
	})
	if strings.Contains(prompt, "--- a ---") {
		t.Fatal("should exclude self position")
	}
	if !strings.Contains(prompt, "--- b ---") {
		t.Fatal("should include peer position")
	}
}

func TestEstimateCouncilCost(t *testing.T) {
	s := estimateCouncilCost(3, 2)
	if !strings.Contains(s, "7 model turns") {
		t.Fatalf("got %q", s)
	}
}

func TestIsCouncilAgentViaDef(t *testing.T) {
	def := model.ADLDefinition{
		Orchestration: &model.ADLOrchestration{
			Type:    model.OrchestrationTypeCouncil,
			Members: []model.ADLOrchestrationMember{{Agent: "claude-code"}},
		},
	}
	if !model.IsCouncilAgent(def) {
		t.Fatal("expected council agent")
	}
}
