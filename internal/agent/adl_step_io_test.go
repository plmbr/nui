// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"strings"
	"testing"

	"nui/internal/model"
)

func TestStepOutputStoreNamedOutputs(t *testing.T) {
	store := newStepOutputStore()
	store.set(model.ADLStep{
		Name: "research",
		Outputs: []model.ADLOutput{
			{Name: "brief", Type: "text"},
		},
	}, "bullet points")

	text, ok := store.resolve("research.brief")
	if !ok || text != "bullet points" {
		t.Fatalf("resolve(research.brief) = %q, %v", text, ok)
	}
}

func TestStepOutputStoreDefaultOutput(t *testing.T) {
	store := newStepOutputStore()
	store.set(model.ADLStep{Name: "research"}, "full output")

	text, ok := store.resolve("research")
	if !ok || text != "full output" {
		t.Fatalf("resolve(research) = %q, %v", text, ok)
	}
}

func TestBuildStepMessageNamedInput(t *testing.T) {
	store := newStepOutputStore()
	store.set(model.ADLStep{
		Name: "research",
		Outputs: []model.ADLOutput{{Name: "brief", Type: "text"}},
	}, "research text")

	msg := buildStepMessage("user prompt", model.ADLStep{
		Inputs: []model.ADLInput{{From: "research.brief", As: "Brief"}},
	}, store)

	if !strings.Contains(msg, "research text") {
		t.Fatalf("message missing research text: %q", msg)
	}
	if !strings.Contains(msg, "## Brief") {
		t.Fatalf("message missing section header: %q", msg)
	}
	if !strings.HasSuffix(strings.TrimSpace(msg), "user prompt") {
		t.Fatalf("message missing user prompt: %q", msg)
	}
}

func TestBuildStepMessageDependsOn(t *testing.T) {
	store := newStepOutputStore()
	store.set(model.ADLStep{Name: "research"}, "dep output")

	msg := buildStepMessage("go", model.ADLStep{
		DependsOn: []string{"research"},
	}, store)

	if !strings.Contains(msg, "dep output") {
		t.Fatalf("message = %q", msg)
	}
	if !strings.Contains(msg, "Output from research") {
		t.Fatalf("message = %q", msg)
	}
}

func TestStepOutputStoreResolveMissing(t *testing.T) {
	store := newStepOutputStore()
	if _, ok := store.resolve("missing.brief"); ok {
		t.Fatal("expected missing ref to fail")
	}
}

func TestStepOutputStoreMultipleOutputNames(t *testing.T) {
	store := newStepOutputStore()
	store.set(model.ADLStep{
		Name: "draft",
		Outputs: []model.ADLOutput{
			{Name: "summary", Type: "text"},
			{Name: "body", Type: "text"},
		},
	}, "same text")

	for _, name := range []string{"draft.summary", "draft.body"} {
		text, ok := store.resolve(name)
		if !ok || text != "same text" {
			t.Fatalf("resolve(%s) = %q, %v", name, text, ok)
		}
	}
}
