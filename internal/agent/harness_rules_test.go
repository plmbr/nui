// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "testing"

func TestRuleFilename(t *testing.T) {
	tests := map[string]string{
		"corp-guidelines": "corp-guidelines.md",
		"Style Guide":     "Style-Guide.md",
		"":                "rule.md",
	}
	for in, want := range tests {
		if got := ruleFilename(in); got != want {
			t.Fatalf("ruleFilename(%q) = %q, want %q", in, got, want)
		}
	}
}
