// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
)

// EnsureSessionWorkspace creates ~/.nui/workspaces/<sessionID> and returns its path.
func EnsureSessionWorkspace(sessionID string) (string, error) {
	if sessionID == "" {
		return "", os.ErrInvalid
	}
	path, err := sessionWorkspacePath(sessionID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return "", err
	}
	return path, nil
}

// RemoveSessionWorkspace deletes ~/.nui/workspaces/<sessionID> if it exists.
func RemoveSessionWorkspace(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	path, err := sessionWorkspacePath(sessionID)
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func sessionWorkspacePath(sessionID string) (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "workspaces", sessionID), nil
}
