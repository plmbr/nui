// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
)

// SessionConfigDir returns ~/.loop/sessions/<sessionID>, creating it if needed.
// Loop provisions per-session harness config (MCP, skills, system prompt) here.
func SessionConfigDir(sessionID string) (string, error) {
	if sessionID == "" {
		return "", os.ErrInvalid
	}
	path, err := sessionConfigDirPath(sessionID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return "", err
	}
	return path, nil
}

func sessionConfigDirPath(sessionID string) (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sessions", sessionID), nil
}

// RemoveSessionConfigDir deletes ~/.loop/sessions/<sessionID> if it exists.
func RemoveSessionConfigDir(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	path, err := sessionConfigDirPath(sessionID)
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}
