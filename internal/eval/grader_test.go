// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package eval

import (
	"strings"
	"testing"

	"loop/internal/model"
)

func TestGraderContains(t *testing.T) {
	g := &Grader{}
	res, err := g.Grade(t.Context(), "Hello, I am an assistant.", &model.ADLEvalExpect{
		Type:  "contains",
		Value: "assistant",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed == nil || !*res.Passed {
		t.Fatal("expected pass")
	}
}

func TestGraderExact(t *testing.T) {
	g := &Grader{}
	res, err := g.Grade(t.Context(), "hello", &model.ADLEvalExpect{
		Type:  "exact",
		Value: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed == nil || !*res.Passed {
		t.Fatal("expected pass")
	}
}

func TestGraderRegex(t *testing.T) {
	g := &Grader{}
	res, err := g.Grade(t.Context(), "answer: 42", &model.ADLEvalExpect{
		Type:  "regex",
		Value: `\d+`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed == nil || !*res.Passed {
		t.Fatal("expected pass")
	}
}

func TestGraderNone(t *testing.T) {
	g := &Grader{}
	res, err := g.Grade(t.Context(), "anything", &model.ADLEvalExpect{Type: "none"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Passed != nil {
		t.Fatal("expected nil passed for none")
	}
}

func TestFilterEvals(t *testing.T) {
	evals := []model.ADLEval{
		{Name: "a", Input: "1"},
		{Name: "b", Input: "2", Disabled: true},
		{Name: "c", Input: "3"},
	}
	all := filterEvals(evals, nil)
	if len(all) != 2 {
		t.Fatalf("got %d enabled evals", len(all))
	}
	one := filterEvals(evals, []string{"c"})
	if len(one) != 1 || one[0].Name != "c" {
		t.Fatalf("got %#v", one)
	}
}

func TestResolveWorkingDir(t *testing.T) {
	wd, err := resolveWorkingDir("/tmp/base", "")
	if err != nil {
		t.Fatal(err)
	}
	if wd != "/tmp/base" {
		t.Fatalf("got %q", wd)
	}
	wd, err = resolveWorkingDir("/tmp/base", "subdir")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(wd, "subdir") {
		t.Fatalf("got %q", wd)
	}
}
