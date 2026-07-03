// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
)

// SessionUploadsDir returns the temp directory for pasted/dropped files in a session.
func SessionUploadsDir(sessionID string) (string, error) {
	dir := filepath.Join(os.TempDir(), "loop-uploads", sessionID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// RemoveSessionUploads deletes all uploaded files for a session.
func RemoveSessionUploads(sessionID string) error {
	dir := filepath.Join(os.TempDir(), "loop-uploads", sessionID)
	return os.RemoveAll(dir)
}
