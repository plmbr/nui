// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package viz

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	chartJSScriptSrcRE   = regexp.MustCompile(`(?i)<script\s+[^>]*src=["']https?://[^"']*chart(\.min)?\.js[^"']*["'][^>]*>\s*</script>`)
	chartCanvasIDRE      = regexp.MustCompile(`getElementById\(\s*['"]([^'"]+)['"]\s*\)\s*\.\s*getContext\(\s*['"]2d['"]\s*\)`)
	chartLabelsRE        = regexp.MustCompile(`['"]?labels['"]?:\s*\[([^\]]+)\]`)
	chartDataRE          = regexp.MustCompile(`['"]?data['"]?:\s*\[([^\]]+)\]`)
	chartBrokenColorRE   = regexp.MustCompile(`rgba\([^)]+\),\s*\d`)
	divIDTagRE           = regexp.MustCompile(`(?i)<div\s+[^>]*\bid\s*=\s*["']([^"']+)["']`)
)

const defaultChartScriptSrc = "/vendor/chart.min.js"

// ScriptsBalanced reports whether every <script> tag has a matching </script>.
func ScriptsBalanced(html string) bool {
	opens := strings.Count(strings.ToLower(html), "<script")
	closes := strings.Count(strings.ToLower(html), "</script>")
	return opens == 0 || opens == closes
}

// PrepareHTML repairs common model output issues before rendering visualizations.
func PrepareHTML(html string) string {
	html = strings.TrimSpace(html)
	if html == "" {
		return html
	}
	html = repairOllamaEscapedHTML(html)
	for scriptCloseCount(html) < scriptOpenCount(html) {
		html += "</script>"
	}
	if shouldRebuildChartHTML(html) {
		if rebuilt := rebuildChartHTML(html, defaultChartScriptSrc); rebuilt != "" {
			html = rebuilt
		}
	} else {
		html = ensureChartCanvas(html)
	}
	html = ensureChartJSLibrary(html, defaultChartScriptSrc)
	lower := strings.ToLower(html)
	if !strings.Contains(lower, "<html") && !strings.Contains(lower, "<!doctype") {
		html = "<!DOCTYPE html><html><head><meta charset=\"utf-8\"></head><body>" + html + "</body></html>"
	}
	html = rewriteChartJSSources(html, defaultChartScriptSrc)
	return injectChartErrorHandler(html)
}

func repairOllamaEscapedHTML(html string) string {
	for pass := 0; pass < 8; pass++ {
		prev := html
		html = strings.ReplaceAll(html, `\\"`, `"`)
		html = strings.ReplaceAll(html, `\"`, `"`)
		html = strings.ReplaceAll(html, `\[`, `[`)
		html = strings.ReplaceAll(html, `\]`, `]`)
		html = strings.ReplaceAll(html, `\\n`, "\n")
		html = strings.ReplaceAll(html, `\\t`, "\t")
		html = strings.ReplaceAll(html, ")_.getContext", ").getContext")
		html = strings.ReplaceAll(html, ")_,", "),")
		html = strings.ReplaceAll(html, `")_,`, `"),`)
		html = strings.ReplaceAll(html, `getContext("2d")_`, `getContext("2d")`)
		html = strings.ReplaceAll(html, `getContext('2d')_`, `getContext('2d')`)
		if html == prev {
			break
		}
	}
	return html
}

func looksOllamaCorrupted(html string) bool {
	lower := strings.ToLower(html)
	return strings.Contains(html, `\\"`) ||
		strings.Contains(html, `\"`) ||
		strings.Contains(html, `\]`) ||
		strings.Contains(html, `\[`) ||
		strings.Contains(html, ")_.") ||
		strings.Contains(lower, `id=\"`) ||
		strings.Contains(lower, `id=\\"`)
}

func scriptOpenCount(html string) int {
	return strings.Count(strings.ToLower(html), "<script")
}

func scriptCloseCount(html string) int {
	return strings.Count(strings.ToLower(html), "</script>")
}

func rewriteChartJSSources(html, chartSrc string) string {
	replacement := fmt.Sprintf(`<script src="%s"></script>`, chartSrc)
	return chartJSScriptSrcRE.ReplaceAllString(html, replacement)
}

func ensureChartCanvas(html string) string {
	if !strings.Contains(strings.ToLower(html), "new chart") {
		return html
	}
	if strings.Contains(strings.ToLower(html), "<canvas") {
		return html
	}
	for _, m := range chartCanvasIDRE.FindAllStringSubmatch(html, -1) {
		if len(m) < 2 {
			continue
		}
		html = replaceDivIDWithCanvas(html, m[1])
	}
	return html
}

func replaceDivIDWithCanvas(html, id string) string {
	double := regexp.MustCompile(`(?i)<div\s+([^>]*\bid\s*=\s*"` + regexp.QuoteMeta(id) + `"[^>]*)\s*>\s*</div>`)
	single := regexp.MustCompile(`(?i)<div\s+([^>]*\bid\s*=\s*'` + regexp.QuoteMeta(id) + `'[^>]*)\s*>\s*</div>`)
	canvas := fmt.Sprintf(`<canvas id="%s" width="400" height="200"></canvas>`, id)
	html = double.ReplaceAllString(html, canvas)
	html = single.ReplaceAllString(html, canvas)
	return html
}

func shouldRebuildChartHTML(html string) bool {
	lower := strings.ToLower(html)
	if !strings.Contains(lower, "new chart") {
		return false
	}
	if looksOllamaCorrupted(html) {
		return true
	}
	if chartBrokenColorRE.MatchString(html) {
		return true
	}
	if strings.Contains(lower, "getcontext('2d')") || strings.Contains(lower, `getcontext("2d")`) {
		for _, m := range divIDTagRE.FindAllStringSubmatch(html, -1) {
			if len(m) < 2 {
				continue
			}
			id := m[1]
			if strings.Contains(html, `getElementById("`+id+`")`) || strings.Contains(html, `getElementById('`+id+`')`) {
				return true
			}
		}
	}
	return false
}

func rebuildChartHTML(html, chartSrc string) string {
	labels := extractChartStringList(chartLabelsRE, html)
	data := extractChartNumberList(chartDataRE, html)
	if len(data) == 0 {
		return ""
	}
	if len(labels) == 0 {
		labels = defaultChartLabels(len(data))
	}
	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return ""
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><canvas id="nui-chart" width="400" height="200"></canvas><script src="%s"></script><script>new Chart(document.getElementById('nui-chart').getContext('2d'),{type:'bar',data:{labels:%s,datasets:[{label:'Series 1',data:%s,backgroundColor:'rgba(54,162,235,0.5)'}]},options:{responsive:false,plugins:{legend:{display:false}}}});</script></body></html>`, chartSrc, labelsJSON, dataJSON)
}

func extractChartStringList(re *regexp.Regexp, html string) []string {
	m := re.FindStringSubmatch(html)
	if len(m) < 2 {
		return nil
	}
	var out []string
	for _, part := range strings.Split(m[1], ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func defaultChartLabels(n int) []string {
	labels := make([]string, n)
	for i := range labels {
		labels[i] = fmt.Sprintf("Item %d", i+1)
	}
	return labels
}

func extractChartNumberList(re *regexp.Regexp, html string) []int {
	m := re.FindStringSubmatch(html)
	if len(m) < 2 {
		return nil
	}
	var out []int
	for _, part := range strings.Split(m[1], ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}

func ensureChartJSLibrary(html, chartSrc string) string {
	lower := strings.ToLower(html)
	if !strings.Contains(lower, "new chart") {
		return html
	}
	if strings.Contains(lower, "chart.min.js") || strings.Contains(lower, "chart.js") {
		return html
	}
	tag := fmt.Sprintf(`<script src="%s"></script>`, chartSrc)
	if idx := strings.Index(lower, "<canvas"); idx >= 0 {
		return html[:idx] + tag + html[idx:]
	}
	if idx := strings.Index(lower, "<script"); idx >= 0 {
		return html[:idx] + tag + html[idx:]
	}
	if idx := strings.Index(lower, "<body"); idx >= 0 {
		close := strings.Index(lower[idx:], ">")
		if close >= 0 {
			insertAt := idx + close + 1
			return html[:insertAt] + tag + html[insertAt:]
		}
	}
	return tag + html
}

const chartErrorHandlerMarker = "Chart failed: '"

var chartInlineScriptRE = regexp.MustCompile(`(?is)<script\b[^>]*>[\s\S]*?new\s+chart[\s\S]*?</script>`)

func injectChartErrorHandler(html string) string {
	if strings.Contains(html, chartErrorHandlerMarker) {
		return html
	}
	handler := `<script>window.addEventListener('error',function(e){var p=document.createElement('pre');p.style.cssText='color:#b91c1c;padding:12px;font:13px/1.4 monospace';p.textContent='Chart failed: '+e.message;document.body.appendChild(p);});</script>`
	if loc := chartInlineScriptRE.FindStringIndex(html); len(loc) == 2 {
		return html[:loc[0]] + handler + html[loc[0]:]
	}
	lower := strings.ToLower(html)
	if idx := strings.LastIndex(lower, "</body>"); idx >= 0 {
		return html[:idx] + handler + html[idx:]
	}
	return html + handler
}
