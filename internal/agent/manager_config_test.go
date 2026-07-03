// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"testing"
)

func TestConfigInt(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want int
	}{
		{name: "int", in: 9090, want: 9090},
		{name: "int64", in: int64(9090), want: 9090},
		{name: "float64 json", in: float64(9090), want: 9090},
		{name: "json number", in: json.Number("9090"), want: 9090},
		{name: "missing", in: nil, want: 0},
		{name: "string", in: "9090", want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := configInt(tc.in); got != tc.want {
				t.Fatalf("configInt(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
