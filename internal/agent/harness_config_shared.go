// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"os"
	"path/filepath"
	"strings"
)

const piAgentSubdir = "pi-agent"

// dockerSessionConfigMount is the in-container path for ~/.loop/sessions/<id>/ bind mounts.
const dockerSessionConfigMount = "/home/loop/.loop/session-config"

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

// userClaudeConfigDir is the default Claude Code config directory (~/.claude).
func userClaudeConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

// linkFileIfMissing creates a symlink at dst pointing to src when src exists and dst is absent.
func linkFileIfMissing(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	return os.Symlink(src, dst)
}
