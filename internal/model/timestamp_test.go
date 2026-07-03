// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import "testing"

func TestNormalizeTimestamp(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"2026-07-02T18:00:00Z", "2026-07-02T18:00:00Z"},
		{"2026-07-02T18:00:00", "2026-07-02T18:00:00Z"},
		{"2026-07-02T18:00:00.000", "2026-07-02T18:00:00Z"},
		{"2026-07-02T11:00:00-07:00", "2026-07-02T18:00:00Z"},
	}
	for _, tt := range tests {
		got := NormalizeTimestamp(tt.in)
		if got != tt.want {
			t.Fatalf("NormalizeTimestamp(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
