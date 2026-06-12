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

// piEntry is a single line from a Pi session JSONL file.
type piEntry struct {
	Type      string     `json:"type"`
	Timestamp string     `json:"timestamp"`
	Message   *piMessage `json:"message"`
}

type piMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // []piContentBlock
}

type piContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// piDirHash converts a working directory path to Pi's session folder name.
// Pi uses: "-" + strings.ReplaceAll(path, "/", "-") + "--"
// e.g. /Users/alice/proj → --Users-alice-proj--
func piDirHash(workingDir string) string {
	return "-" + strings.ReplaceAll(workingDir, "/", "-") + "--"
}

// piSessionFile finds the .jsonl file for the given session ID under Pi's sessions directory.
func piSessionFile(workingDir, sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".pi", "agent", "sessions", piDirHash(workingDir))
	pattern := filepath.Join(dir, "*_"+sessionID+".jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", os.ErrNotExist
	}
	return matches[0], nil
}

// LoadPiHistory reads the Pi session JSONL for a project and returns
// user/assistant messages in chronological order.
func LoadPiHistory(workingDir, sessionID string) ([]model.ChatMessage, error) {
	if sessionID == "" {
		return []model.ChatMessage{}, nil
	}

	if workingDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return []model.ChatMessage{}, nil
		}
		workingDir = cwd
	}

	filePath, err := piSessionFile(workingDir, sessionID)
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
		var entry piEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Type != "message" || entry.Message == nil {
			continue
		}

		content := extractPiContent(entry.Message)
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

// DeletePiSession removes the Pi session .jsonl file for the given session.
func DeletePiSession(workingDir, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	if workingDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil
		}
		workingDir = cwd
	}
	filePath, err := piSessionFile(workingDir, sessionID)
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

func extractPiContent(m *piMessage) string {
	raw := strings.TrimSpace(string(m.Content))
	if raw == "" || raw[0] != '[' {
		return ""
	}
	var blocks []piContentBlock
	if err := json.Unmarshal(m.Content, &blocks); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return strings.TrimSpace(sb.String())
}
