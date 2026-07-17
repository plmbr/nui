// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
)

// runsDirOverride is set in tests to avoid writing under the real home directory.
var runsDirOverride string

// SetRunsDirOverride directs run logs to dir (tests only).
func SetRunsDirOverride(dir string) {
	runsDirOverride = dir
}

// RunsDir returns ~/.nui/runs for durable run event logs.
func RunsDir() (string, error) {
	if runsDirOverride != "" {
		if err := os.MkdirAll(runsDirOverride, 0700); err != nil {
			return "", err
		}
		return runsDirOverride, nil
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	runsDir := filepath.Join(dir, "runs")
	if err := os.MkdirAll(runsDir, 0700); err != nil {
		return "", err
	}
	return runsDir, nil
}

// RunLogPath returns ~/.nui/runs/<runID>.jsonl.
func RunLogPath(runID string) (string, error) {
	dir, err := RunsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, runID+".jsonl"), nil
}
