// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"loop/internal/model"
)

// claudeEntry is a single line from a Claude Code session JSONL file.
type claudeEntry struct {
	Type       string          `json:"type"`
	IsSidechain bool           `json:"isSidechain"`
	Timestamp  string          `json:"timestamp"`
	Message    *claudeMessage  `json:"message"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // string for user, []block for assistant
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// LoadClaudeHistory reads the Claude Code session JSONL for a project and
// returns user/assistant messages in chronological order.
func LoadClaudeHistory(workingDir, sessionID string) ([]model.ChatMessage, error) {
	if sessionID == "" {
		return []model.ChatMessage{}, nil
	}

	// When workingDir is empty the agent inherits the server's CWD — mirror that here.
	if workingDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return []model.ChatMessage{}, nil
		}
		workingDir = cwd
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Claude encodes the working directory by replacing every "/" with "-".
	// e.g. /Users/alice/proj → -Users-alice-proj
	dirHash := strings.ReplaceAll(workingDir, "/", "-")
	filePath := filepath.Join(home, ".claude", "projects", dirHash, sessionID+".jsonl")

	f, err := os.Open(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return []model.ChatMessage{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var msgs []model.ChatMessage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry claudeEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		// Skip subagent turns and non-message entries.
		if entry.IsSidechain || entry.Message == nil {
			continue
		}
		if entry.Type != "user" && entry.Type != "assistant" {
			continue
		}

		content := extractContent(entry.Message)
		if content == "" {
			continue
		}

		ts := entry.Timestamp
		if ts == "" {
			ts = time.Now().UTC().Format(time.RFC3339)
		}

		msgs = append(msgs, model.ChatMessage{
			ID:        uuid.NewString(),
			Role:      entry.Message.Role,
			Content:   content,
			CreatedAt: ts,
		})
	}
	if err := scanner.Err(); err != nil {
		return msgs, err
	}
	if msgs == nil {
		msgs = []model.ChatMessage{}
	}
	return msgs, nil
}

func extractContent(m *claudeMessage) string {
	raw := strings.TrimSpace(string(m.Content))
	if raw == "" {
		return ""
	}

	// User messages: content is a plain JSON string.
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(m.Content, &s); err == nil {
			return strings.TrimSpace(s)
		}
	}

	// Assistant messages: content is an array of typed blocks.
	if raw[0] == '[' {
		var blocks []claudeContentBlock
		if err := json.Unmarshal(m.Content, &blocks); err == nil {
			var sb strings.Builder
			for _, b := range blocks {
				if b.Type == "text" {
					sb.WriteString(b.Text)
				}
			}
			return strings.TrimSpace(sb.String())
		}
	}

	return ""
}
