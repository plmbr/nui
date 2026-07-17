// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package viz

import "testing"

func TestIsVisualizationTool(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"show_visualization", true},
		{"mcp__nui-viz__show_visualization", true},
		{"nui-viz:show_visualization", true},
		{"ask_user", false},
		{"mcp__nui-hitl__ask_user", false},
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

func TestVisualizationHTMLReady(t *testing.T) {
	partial := `<canvas id="chart">`
	if VisualizationHTMLReady(partial) {
		t.Fatal("partial canvas should not be ready")
	}
	ready := `<canvas id="c"></canvas><script>new Chart(document.getElementById("c"))</script>`
	if !VisualizationHTMLReady(ready) {
		t.Fatal("complete canvas chart should be ready")
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="120"><rect width="120" height="120"/></svg>`
	if !VisualizationHTMLReady(svg) {
		t.Fatal("closed svg should be ready")
	}
	emptyShell := `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body></body></html>`
	if VisualizationHTMLReady(emptyShell) {
		t.Fatal("empty html shell should not be ready")
	}
	proseOnly := `<!DOCTYPE html><html><body><p>France's capital is Paris.</p></body></html>`
	if VisualizationHTMLReady(proseOnly) {
		t.Fatal("prose-only html should not be ready")
	}
}

func TestPlainTextFromHTML(t *testing.T) {
	html := `<!DOCTYPE html><html><body><p>France's capital is Paris.</p></body></html>`
	if got := PlainTextFromHTML(html); got != "France's capital is Paris." {
		t.Fatalf("PlainTextFromHTML() = %q", got)
	}
}
