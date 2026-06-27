// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"~", home},
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"/tmp/abs", "/tmp/abs"},
		{"relative/path", "relative/path"},
	}
	for _, tc := range tests {
		got, err := ExpandHome(tc.in)
		if err != nil {
			t.Fatalf("ExpandHome(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ExpandHome(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
