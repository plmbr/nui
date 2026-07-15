// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package viz

import (
	"strings"

	"loop/internal/model"
)

// HTMLMatches reports whether two HTML documents are likely the same visualization.
func HTMLMatches(a, b string) bool {
	na := normalizeHTML(a)
	nb := normalizeHTML(b)
	if na == "" || nb == "" {
		return false
	}
	if na == nb {
		return true
	}
	shorter, longer := na, nb
	if len(na) > len(nb) {
		shorter, longer = nb, na
	}
	if len(shorter) < 100 {
		return false
	}
	if strings.Contains(longer, shorter) {
		return true
	}
	prefixLen := min(200, len(shorter))
	if shorter[:prefixLen] != longer[:prefixLen] {
		return false
	}
	diff := len(longer) - len(shorter)
	if diff < 0 {
		diff = -diff
	}
	return diff*100/len(longer) <= 2
}

func normalizeHTML(s string) string {
	return strings.Map(func(r rune) rune {
		if r <= ' ' {
			return -1
		}
		return r
	}, strings.ToLower(strings.TrimSpace(s)))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NormalizeParts enriches visualizationHtml on tool parts and removes duplicate Write copies
// when show_visualization already carries the same chart in the same message.
func NormalizeParts(parts []model.ChatMessagePart) []model.ChatMessagePart {
	if len(parts) == 0 {
		return parts
	}
	out := make([]model.ChatMessagePart, len(parts))
	copy(out, parts)

	var showVizHTML []string
	for i := range out {
		if out[i].Type != "tool" {
			continue
		}
		if out[i].VisualizationHTML == "" {
			if html, title, ok := ParseFromTool(out[i].ToolName, out[i].ToolArgs); ok {
				out[i].VisualizationHTML = html
				out[i].VisualizationTitle = title
			}
		}
		if IsVisualizationTool(out[i].ToolName) && out[i].VisualizationHTML != "" {
			showVizHTML = append(showVizHTML, out[i].VisualizationHTML)
		}
	}

	for i := range out {
		if out[i].Type != "tool" || out[i].VisualizationHTML == "" {
			continue
		}
		if IsInlineHTMLTool(out[i].ToolName) {
			for _, sv := range showVizHTML {
				if HTMLMatches(out[i].VisualizationHTML, sv) {
					out[i].VisualizationHTML = ""
					out[i].VisualizationTitle = ""
					break
				}
			}
		}
	}

	seenByToolCall := make(map[string]string)
	for i := range out {
		html := out[i].VisualizationHTML
		if html == "" {
			continue
		}
		dup := false
		if tcID := strings.TrimSpace(out[i].ToolCallID); tcID != "" {
			if prev, ok := seenByToolCall[tcID]; ok && HTMLMatches(html, prev) {
				dup = true
			} else {
				seenByToolCall[tcID] = html
			}
		} else {
			for _, prev := range seenByToolCall {
				if HTMLMatches(html, prev) {
					dup = true
					break
				}
			}
			if !dup {
				seenByToolCall[out[i].ID] = html
			}
		}
		if dup {
			out[i].VisualizationHTML = ""
			out[i].VisualizationTitle = ""
		}
	}
	return out
}
