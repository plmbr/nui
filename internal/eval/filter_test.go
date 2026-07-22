// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package eval

import (
	"testing"

	"nui/internal/model"
)

func TestFilterEvals_skipsDisabledNamedCases(t *testing.T) {
	evals := []model.ADLEval{
		{Name: "enabled", Input: "hi"},
		{Name: "disabled", Input: "bye", Disabled: true},
	}
	out := filterEvals(evals, []string{"enabled", "disabled"})
	if len(out) != 1 || out[0].Name != "enabled" {
		t.Fatalf("filterEvals() = %+v, want only enabled", out)
	}
}
