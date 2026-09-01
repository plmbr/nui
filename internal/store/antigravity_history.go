// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"

	"nui/internal/model"
)

// Antigravity conversations live under ~/.gemini/antigravity-cli/conversations/<uuid>.db
// (opaque/encrypted). nui resumes via conversation id but does not parse transcripts yet.

func antigravityConversationsDir() (string, error) {
	if root := os.Getenv("ANTIGRAVITY_CLI_ROOT"); root != "" {
		return filepath.Join(root, "conversations"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "antigravity-cli", "conversations"), nil
}

// LoadAntigravityHistory returns an empty transcript. Antigravity stores conversations
// in opaque SQLite/protobuf files that nui cannot decode without a live daemon.
func LoadAntigravityHistory(_ string, _ string) ([]model.ChatMessage, error) {
	return []model.ChatMessage{}, nil
}

// DeleteAntigravitySession removes the conversation database for the given conversation id.
func DeleteAntigravitySession(_ string, conversationID string) error {
	if conversationID == "" {
		return nil
	}
	dir, err := antigravityConversationsDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, conversationID+".db")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Best-effort cleanup of SQLite sidecars.
	_ = os.Remove(path + "-wal")
	_ = os.Remove(path + "-shm")
	_ = os.Remove(filepath.Join(dir, conversationID+".pb"))
	return nil
}
