// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mentions

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// The quoting and punctuation rules match Chatbook mentions in
// notebook-intelligence (chatbook_mentions.py, plmbr/notebook-intelligence#503),
// which follow Claude Code and Codex; change the two together. Unlike Chatbook,
// a mention here must follow whitespace, as Claude Code requires of the `@path`
// mentions nui hands it (see findMentionTokens).

// trailingPunctuation is the sentence punctuation that, typed after an unquoted
// file or dir mention (`see @file:notes.md,`), is not part of the path.
const trailingPunctuation = ".,;:!?)]}'\"\u2026\u201d\u2019\u00bb" +
	"\u3002\u3001\uff0c\uff1b\uff1a\uff01\uff1f\uff09\u300d\u300f"

// maxTrailingPunctuation caps how many of those characters are trimmed.
const maxTrailingPunctuation = 5

// mentionToken is one `@...` mention found in a message.
type mentionToken struct {
	start, end int    // byte offsets of the whole token, `@` included
	value      string // mention value with any quotes removed, e.g. "file:a b.md"
	quoted     bool
}

// endsUnquotedToken reports whether r ends an unquoted mention: whitespace,
// the file, group, record, and unit separators Python also treats as
// whitespace, or `@`.
func endsUnquotedToken(r rune) bool {
	return r == '@' || unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)
}

// breaksQuotedToken reports whether r cannot appear inside a quoted mention.
func breaksQuotedToken(r rune) bool {
	switch r {
	case '"', '\n', '\r', '\u0085', '\u2028', '\u2029':
		return true
	}
	return false
}

// isWordRune matches Python's Unicode \w, which Chatbook's tokenizer uses.
func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// findMentionTokens returns the mentions in message, in order. A mention starts
// at an `@` at the start of the message or after whitespace, which is also what
// Claude Code requires of the `@path` mentions nui hands it. It is either
// quoted, `@file:"data/my notes.md"`, which is how a value containing
// whitespace or `@` is written, or unquoted and runs to the next whitespace or
// `@`. A closing quote followed directly by a word character or another quote
// does not end a quoted mention; the text is read as one unquoted token, as it
// was before quoting existed.
func findMentionTokens(message string) []mentionToken {
	var tokens []mentionToken
	for i := 0; i < len(message); {
		offset := strings.IndexByte(message[i:], '@')
		if offset < 0 {
			break
		}
		start := i + offset
		if start > 0 {
			prev, _ := utf8.DecodeLastRuneInString(message[:start])
			if !unicode.IsSpace(prev) {
				i = start + 1
				continue
			}
		}
		if token, ok := quotedMentionAt(message, start); ok {
			tokens = append(tokens, token)
			i = token.end
			continue
		}
		end := start + 1
		for end < len(message) {
			r, size := utf8.DecodeRuneInString(message[end:])
			if endsUnquotedToken(r) {
				break
			}
			end += size
		}
		if end > start+1 {
			tokens = append(tokens, mentionToken{start: start, end: end, value: message[start+1 : end]})
		}
		i = end
	}
	return tokens
}

func quotedMentionAt(message string, start int) (mentionToken, bool) {
	rest := message[start+1:]
	for _, kind := range []string{"file", "dir", "ext"} {
		prefix := kind + `:"`
		if !strings.HasPrefix(rest, prefix) {
			continue
		}
		body := rest[len(prefix):]
		closing := strings.IndexFunc(body, breaksQuotedToken)
		if closing <= 0 || body[closing] != '"' {
			return mentionToken{}, false
		}
		end := start + 1 + len(prefix) + closing + 1
		if next, size := utf8.DecodeRuneInString(message[end:]); size > 0 && (next == '"' || isWordRune(next)) {
			return mentionToken{}, false
		}
		return mentionToken{
			start:  start,
			end:    end,
			value:  kind + ":" + body[:closing],
			quoted: true,
		}, true
	}
	return mentionToken{}, false
}

// pathCandidates returns value, then value with trailing punctuation removed
// one character at a time, at most maxTrailingPunctuation times and never to
// an empty path. The caller stops at the first candidate that exists, so a
// name that really ends in punctuation still resolves and a shorter, unrelated
// name is never reached past it.
func pathCandidates(prefix, value string) []string {
	candidates := []string{value}
	for len(candidates) <= maxTrailingPunctuation {
		last := candidates[len(candidates)-1]
		r, size := utf8.DecodeLastRuneInString(last)
		if size == 0 || len(last)-size <= len(prefix) || !strings.ContainsRune(trailingPunctuation, r) {
			break
		}
		candidates = append(candidates, last[:len(last)-size])
	}
	return candidates
}

// atPath renders a resolved absolute path as a mention for the agent, quoting
// it the way Claude Code expects when it contains whitespace.
func atPath(abs string) string {
	if strings.IndexFunc(abs, unicode.IsSpace) >= 0 {
		return `@"` + abs + `"`
	}
	return "@" + abs
}
