// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agentindex

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

// Doc is the searchable text representation of an agent.
type Doc struct {
	ID   string
	Text string
	Hash string
}

// BuildDoc builds a stable document from agent metadata.
func BuildDoc(id, label, description string, tags []string) Doc {
	id = strings.TrimSpace(id)
	parts := make([]string, 0, 4+len(tags))
	if local := agentLocalID(id); local != "" {
		parts = append(parts, local)
	}
	if label = strings.TrimSpace(label); label != "" {
		parts = append(parts, label)
	}
	if description = strings.TrimSpace(description); description != "" {
		parts = append(parts, description)
	}
	for _, tag := range tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			parts = append(parts, tag)
		}
	}
	text := strings.Join(parts, "\n")
	sum := sha256.Sum256([]byte(text))
	return Doc{
		ID:   id,
		Text: text,
		Hash: hex.EncodeToString(sum[:]),
	}
}

func agentLocalID(id string) string {
	id = strings.TrimSpace(id)
	if idx := strings.LastIndex(id, "/"); idx >= 0 {
		return id[idx+1:]
	}
	return id
}

// TokenizeWithFreq returns term frequencies (not deduped).
func TokenizeWithFreq(text string) map[string]float64 {
	tf := map[string]float64{}
	var b strings.Builder
	flush := func() {
		tok := strings.ToLower(b.String())
		b.Reset()
		if len(tok) < 2 || isStopword(tok) {
			return
		}
		tf[tok]++
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return tf
}

func isStopword(tok string) bool {
	switch tok {
	case "a", "an", "the", "and", "or", "to", "of", "in", "on", "for", "is", "are",
		"be", "as", "at", "by", "it", "with", "from", "that", "this", "can", "use",
		"using", "into", "about", "agent", "help", "helpful", "assistant":
		return true
	default:
		return false
	}
}
