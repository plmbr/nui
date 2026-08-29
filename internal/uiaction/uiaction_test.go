// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package uiaction

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		a    Action
		want string
	}{
		{Action{Type: TypeNavigate, Target: TargetCustomize}, ""},
		{Action{Type: TypeNavigate, Target: TargetNewSession}, ""},
		{Action{Type: TypeSetTheme, Theme: ThemeDark}, ""},
		{Action{Type: TypeRefreshUI}, ""},
		{Action{Type: TypeNavigate, Target: "nope"}, "navigate target"},
		{Action{Type: TypeSetTheme, Theme: "blue"}, "set_theme theme"},
		{Action{Type: "click"}, "unknown"},
	}
	for _, tc := range cases {
		got := Validate(tc.a)
		if tc.want == "" {
			if got != "" {
				t.Fatalf("Validate(%+v) = %q, want empty", tc.a, got)
			}
			continue
		}
		if !contains(got, tc.want) {
			t.Fatalf("Validate(%+v) = %q, want containing %q", tc.a, got, tc.want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
