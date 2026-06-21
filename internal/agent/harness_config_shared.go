// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"path/filepath"
	"strings"
)

const piAgentSubdir = "pi-agent"

// harnessConfigBindDir returns the filesystem path bound into sandboxes and pointed at by
// harness config env vars. Pi uses a nested agent leaf directory.
func harnessConfigBindDir(harnessType, sessionConfigDir string) string {
	if sessionConfigDir == "" {
		return ""
	}
	if normalizeHarnessType(harnessType) == "pi" {
		return piAgentConfigDir(sessionConfigDir)
	}
	return sessionConfigDir
}

func piAgentConfigDir(sessionConfigDir string) string {
	return filepath.Join(sessionConfigDir, piAgentSubdir)
}

// escapeTOMLMultiline prepares content for a TOML """ block.
func escapeTOMLMultiline(s string) string {
	return strings.ReplaceAll(s, "\\", "\\\\")
}
