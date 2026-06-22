// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"fmt"
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

// codexTOMLInlineTable formats a map as a TOML inline table, e.g. { KEY = "value" }.
func codexTOMLInlineTable(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	items := make([]string, 0, len(m))
	for k, v := range m {
		items = append(items, fmt.Sprintf("%s = %q", codexTOMLKey(k), v))
	}
	return "{" + strings.Join(items, ", ") + "}"
}

func codexTOMLKey(k string) string {
	for _, r := range k {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return fmt.Sprintf("%q", k)
		}
	}
	return k
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
