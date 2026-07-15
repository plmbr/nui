// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package viz

import "strings"

// PlainTextFromHTML extracts visible text from visualization HTML for plain-text fallback.
func PlainTextFromHTML(html string) string {
	body := extractBodyInnerHTML(html)
	body = stripScriptBlocks(body)
	return strings.TrimSpace(stripHTMLTags(body))
}
