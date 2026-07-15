// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package viz

import "strings"

const (
	MCPName  = "loop-viz"
	ToolName = "show_visualization"
)

// IsVisualizationTool reports whether toolName refers to the show_visualization tool.
func IsVisualizationTool(toolName string) bool {
	return BareToolName(toolName) == ToolName
}

// IsInlineHTMLTool reports whether toolName is a harness write/edit tool that may carry HTML.
func IsInlineHTMLTool(toolName string) bool {
	switch BareToolName(toolName) {
	case "Write", "Edit", "write_file", "create_file":
		return true
	default:
		return false
	}
}

// BareToolName strips harness-specific MCP prefixes from a tool name.
func BareToolName(toolName string) string {
	toolName = strings.TrimSpace(toolName)
	if toolName == ToolName {
		return ToolName
	}
	for _, sep := range []string{"__", ":", "/"} {
		if idx := strings.LastIndex(toolName, sep); idx >= 0 {
			bare := toolName[idx+len(sep):]
			if bare == ToolName {
				return ToolName
			}
		}
	}
	return toolName
}

// VisualizationHTMLReady reports whether HTML is complete enough to render or execute.
func VisualizationHTMLReady(html string) bool {
	html = strings.TrimSpace(html)
	if len(html) < 40 {
		return false
	}
	lower := strings.ToLower(html)
	if strings.Contains(lower, "<canvas") {
		if !strings.Contains(lower, "</canvas>") {
			return false
		}
		if !strings.Contains(lower, "new chart") && !strings.Contains(lower, "getcontext(") {
			return false
		}
		if strings.Contains(lower, "<script") {
			if !ScriptsBalanced(html) {
				return false
			}
		}
		return true
	}
	if strings.Contains(lower, "<svg") {
		return strings.Contains(lower, "</svg>")
	}
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype") {
		return strings.Contains(lower, "</html>")
	}
	return looksLikeHTML(html)
}

// ParseInput extracts visualization HTML and optional title from show_visualization arguments.
func ParseInput(args map[string]any) (html, title string, ok bool) {
	if args == nil {
		return "", "", false
	}
	html, _ = args["html"].(string)
	html = strings.TrimSpace(html)
	if html == "" {
		return "", "", false
	}
	title, _ = args["title"].(string)
	return html, strings.TrimSpace(title), true
}

// ParseFromTool extracts inline visualization HTML from tool args for supported tools.
func ParseFromTool(toolName string, args map[string]any) (html, title string, ok bool) {
	if html, title, ok = ParseInput(args); ok && IsVisualizationTool(toolName) {
		return html, title, true
	}
	if !IsInlineHTMLTool(toolName) {
		return "", "", false
	}
	content, _ := args["content"].(string)
	content = strings.TrimSpace(content)
	if !looksLikeHTML(content) {
		return "", "", false
	}
	title = titleFromWriteArgs(args)
	return content, title, true
}

func looksLikeHTML(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "<!doctype") ||
		strings.HasPrefix(lower, "<html") ||
		strings.Contains(lower, "<canvas") ||
		strings.Contains(lower, "<svg")
}

func titleFromWriteArgs(args map[string]any) string {
	for _, key := range []string{"file_path", "filePath", "path"} {
		if v, _ := args[key].(string); v != "" {
			base := v
			if idx := strings.LastIndexAny(v, "/\\"); idx >= 0 {
				base = v[idx+1:]
			}
			if ext := strings.LastIndex(base, "."); ext > 0 {
				base = base[:ext]
			}
			base = strings.ReplaceAll(base, "_", " ")
			base = strings.ReplaceAll(base, "-", " ")
			if base != "" {
				return strings.TrimSpace(base)
			}
		}
	}
	return ""
}
