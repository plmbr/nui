// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import "testing"

func TestValidateADLEvals(t *testing.T) {
	t.Run("empty is valid", func(t *testing.T) {
		if err := ValidateADLEvals(nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("single input with contains", func(t *testing.T) {
		err := ValidateADLEvals([]ADLEval{{
			Name:  "greet",
			Input: "Hello",
			Expect: &ADLEvalExpect{
				Type:  "contains",
				Value: "hi",
			},
		}})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("multi-turn messages", func(t *testing.T) {
		err := ValidateADLEvals([]ADLEval{{
			Name: "math",
			Messages: []ADLEvalMessage{
				{Role: "user", Content: "2+2?"},
				{Role: "assistant", Content: "4"},
				{Role: "user", Content: "sure?"},
			},
			Expect: &ADLEvalExpect{Type: "none"},
		}})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("requires name", func(t *testing.T) {
		err := ValidateADLEvals([]ADLEval{{Input: "hi"}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("duplicate names", func(t *testing.T) {
		err := ValidateADLEvals([]ADLEval{
			{Name: "a", Input: "1"},
			{Name: "a", Input: "2"},
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("input and messages mutually exclusive", func(t *testing.T) {
		err := ValidateADLEvals([]ADLEval{{
			Name:     "both",
			Input:    "hi",
			Messages: []ADLEvalMessage{{Role: "user", Content: "hi"}},
		}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("messages must end with user", func(t *testing.T) {
		err := ValidateADLEvals([]ADLEval{{
			Name: "bad",
			Messages: []ADLEvalMessage{
				{Role: "user", Content: "hi"},
				{Role: "assistant", Content: "hello"},
			},
		}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid regex", func(t *testing.T) {
		err := ValidateADLEvals([]ADLEval{{
			Name:  "rx",
			Input: "hi",
			Expect: &ADLEvalExpect{
				Type:  "regex",
				Value: "[",
			},
		}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("llm requires criteria", func(t *testing.T) {
		err := ValidateADLEvals([]ADLEval{{
			Name:   "judge",
			Input:  "hi",
			Expect: &ADLEvalExpect{Type: "llm"},
		}})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestEffectiveEvalTimeout(t *testing.T) {
	def := ADLDefinition{Harness: ADLHarness{Type: "claude-code"}}
	if got := EffectiveEvalTimeout(def, ADLEval{}); got != 120 {
		t.Fatalf("got %d want 120", got)
	}
	if got := EffectiveEvalTimeout(def, ADLEval{Timeout: 60}); got != 60 {
		t.Fatalf("got %d want 60", got)
	}
	dev := ADLDefinition{Harness: ADLHarness{Type: "devcontainer", InnerHarness: "claude-code"}}
	if got := EffectiveEvalTimeout(dev, ADLEval{}); got != 300 {
		t.Fatalf("got %d want 300", got)
	}
}
