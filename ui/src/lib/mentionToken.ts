// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

// Matches the server's mention tokenizer (internal/mentions/tokens.go) and
// Chatbook mentions in notebook-intelligence (plmbr/notebook-intelligence#503).

/**
 * Whether an unquoted token would end partway through `text`: at whitespace,
 * at `@`, or at U+001C to U+001F, which the server also treats as separators.
 */
function endsUnquotedToken(text: string): boolean {
  return /[\s@]/u.test(text) || [...text].some((ch) => ch >= '\u001c' && ch <= '\u001f')
}

/**
 * The token to insert for a picked mention value, quoted as
 * `@file:"my notes.md"` when an unquoted token would cut it short, the way
 * Codex quotes a picked path. The quoted form cannot hold a quote or a line
 * break, so such a value is left as is.
 */
export function mentionToken(value: string): string {
  const match = /^(file|dir|ext):(.*)$/su.exec(value)
  if (!match || !endsUnquotedToken(match[2]) || /["\n\r\u0085\u2028\u2029]/u.test(match[2])) {
    return `@${value}`
  }
  return `@${match[1]}:"${match[2]}"`
}
