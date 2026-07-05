// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package viz

import "testing"

func TestIsVisualizationTool(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"show_visualization", true},
		{"mcp__loop-viz__show_visualization", true},
		{"loop-viz:show_visualization", true},
		{"ask_user", false},
		{"mcp__loop-hitl__ask_user", false},
	}
	for _, tc := range cases {
		if got := IsVisualizationTool(tc.name); got != tc.want {
			t.Errorf("IsVisualizationTool(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestParseInput(t *testing.T) {
	html, title, ok := ParseInput(map[string]any{
		"html":  " <div>chart</div> ",
		"title": " Sales ",
	})
	if !ok || html != "<div>chart</div>" || title != "Sales" {
		t.Fatalf("ParseInput() = (%q, %q, %v)", html, title, ok)
	}
	if _, _, ok := ParseInput(map[string]any{"title": "x"}); ok {
		t.Fatal("expected missing html to fail")
	}
}

func TestParseFromTool_writeHTML(t *testing.T) {
	html := "<!DOCTYPE html><html><body><canvas></canvas></body></html>"
	got, title, ok := ParseFromTool("Write", map[string]any{
		"content":   html,
		"file_path": "/tmp/ca_ethnicity_pie.html",
	})
	if !ok || got != html || title != "ca ethnicity pie" {
		t.Fatalf("ParseFromTool(Write) = (%q, %q, %v)", got, title, ok)
	}
}
