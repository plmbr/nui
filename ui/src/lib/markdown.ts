// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

/** Normalize agent markdown so inline code delimiters parse reliably. */
export function normalizeMarkdown(content: string): string {
  return content
    .replace(/[\u02CB\uFF40\u2035\u2032]/g, '`')
    .replace(/\\`/g, '`')
    .replace(/``\s*`([^`\n]+)`\s*``/g, '`$1`')
    .replace(/`\s*`([^`\n]+)`(?:\s*`)?/g, '`$1`')
}

/** Strip stray delimiter backticks that ended up inside inline code text. */
export function stripInlineCodeDelimiters(text: string): string {
  return text.replace(/^`+|`+$/g, '')
}
