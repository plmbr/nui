// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"loop/internal/model"
)

type opencodeExport struct {
	Messages []opencodeMessage `json:"messages"`
}

type opencodeMessage struct {
	Info  opencodeMessageInfo  `json:"info"`
	Parts []opencodeMessagePart `json:"parts"`
}

type opencodeMessageInfo struct {
	Role string `json:"role"`
	Time struct {
		Created int64 `json:"created"`
	} `json:"time"`
}

type opencodeMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// LoadOpenCodeHistory runs `opencode export <sessionID>` and returns
// user/assistant messages in chronological order.
func LoadOpenCodeHistory(_, sessionID string) ([]model.ChatMessage, error) {
	if sessionID == "" {
		return []model.ChatMessage{}, nil
	}

	out, err := exec.Command("opencode", "export", sessionID).Output()
	if err != nil {
		return []model.ChatMessage{}, nil
	}

	var export opencodeExport
	if err := json.Unmarshal(out, &export); err != nil {
		return []model.ChatMessage{}, nil
	}

	var msgs []model.ChatMessage
	for _, m := range export.Messages {
		if m.Info.Role != "user" && m.Info.Role != "assistant" {
			continue
		}
		text := extractOpenCodeText(m.Parts)
		if text == "" {
			continue
		}
		ts := time.Now().UTC().Format(time.RFC3339)
		if m.Info.Time.Created > 0 {
			ts = time.UnixMilli(m.Info.Time.Created).UTC().Format(time.RFC3339)
		}
		msgs = append(msgs, model.ChatMessage{
			ID:        uuid.NewString(),
			Role:      m.Info.Role,
			Content:   text,
			CreatedAt: ts,
		})
	}
	if msgs == nil {
		msgs = []model.ChatMessage{}
	}
	return msgs, nil
}

// DeleteOpenCodeSession runs `opencode session delete <sessionID>`.
func DeleteOpenCodeSession(_, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return exec.Command("opencode", "session", "delete", sessionID).Run()
}

func extractOpenCodeText(parts []opencodeMessagePart) string {
	var sb strings.Builder
	for _, p := range parts {
		if p.Type == "text" {
			sb.WriteString(p.Text)
		}
	}
	return strings.TrimSpace(sb.String())
}
