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

// codexEntry is a single line from a Codex session JSONL file.
type codexEntry struct {
	Timestamp string       `json:"timestamp"`
	Type      string       `json:"type"`
	Payload   codexPayload `json:"payload"`
}

type codexPayload struct {
	Type    string             `json:"type"`
	Role    string             `json:"role"`
	Content []codexContentBlock `json:"content"`
}

type codexContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// codexSessionFile finds the .jsonl file for the given thread ID under Codex's
// sessions directory (~/.codex/sessions/YYYY/MM/DD/*-<threadID>.jsonl).
func codexSessionFile(sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sessionsDir := filepath.Join(home, ".codex", "sessions")
	suffix := sessionID + ".jsonl"
	var found string
	_ = filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), suffix) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return "", os.ErrNotExist
	}
	return found, nil
}

// LoadCodexHistory reads the Codex session JSONL for a thread and returns
// user/assistant messages in chronological order.
func LoadCodexHistory(_, sessionID string) ([]model.ChatMessage, error) {
	if sessionID == "" {
		return []model.ChatMessage{}, nil
	}

	filePath, err := codexSessionFile(sessionID)
	if errors.Is(err, os.ErrNotExist) {
		return []model.ChatMessage{}, nil
	}
	if err != nil {
		return nil, err
	}

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
		var entry codexEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Type != "response_item" {
			continue
		}
		p := entry.Payload
		if p.Type != "message" || (p.Role != "user" && p.Role != "assistant") {
			continue
		}

		text := extractCodexContent(p.Content)
		if text == "" {
			continue
		}

		// Codex prepends an <environment_context> block to every user turn — skip it.
		if p.Role == "user" && strings.HasPrefix(text, "<environment_context>") {
			continue
		}

		ts := entry.Timestamp
		if ts == "" {
			ts = time.Now().UTC().Format(time.RFC3339)
		}

		msgs = append(msgs, model.ChatMessage{
			ID:        uuid.NewString(),
			Role:      p.Role,
			Content:   text,
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

// DeleteCodexSession removes the Codex session .jsonl file for the given thread.
func DeleteCodexSession(_, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	filePath, err := codexSessionFile(sessionID)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return nil
	}
	err = os.Remove(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func extractCodexContent(blocks []codexContentBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "input_text" || b.Type == "output_text" {
			sb.WriteString(b.Text)
		}
	}
	return strings.TrimSpace(sb.String())
}
