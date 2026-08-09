// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const piAgentSubdir = "pi-agent"

// dockerSessionConfigMount is the in-container path for ~/.nui/sessions/<id>/ bind mounts.
const dockerSessionConfigMount = "/home/nui/.nui/session-config"

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

// userPiAgentDir is the default Pi agent directory (~/.pi/agent), the value
// PI_CODING_AGENT_DIR normally points at.
func userPiAgentDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pi", "agent"), nil
}

// userOpenCodeConfigDir is the default OpenCode config directory, honoring XDG_CONFIG_HOME.
func userOpenCodeConfigDir() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "opencode"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "opencode"), nil
}

// seedsUserConfig reports whether the harness reads the session config dir from the host
// filesystem. Container sandboxes mount the dir into an image with its own filesystem,
// where symlinks into the user's home would dangle.
func (d HarnessDeps) seedsUserConfig() bool {
	switch normalizeSandbox(d.Sandbox) {
	case "docker", sandboxDevcontainer:
		return false
	default:
		return true
	}
}

// linkUserConfigEntries symlinks named entries from a harness's user config directory into
// an isolated session config directory, so redirecting the harness config env var does not
// hide credentials, provider settings, or installed plugins. Entries nui already generated
// are left untouched.
func linkUserConfigEntries(srcDir, dstDir string, names []string) error {
	same, err := sameDir(srcDir, dstDir)
	if err != nil || same {
		return err
	}
	for _, name := range names {
		if err := linkFileIfMissing(filepath.Join(srcDir, name), filepath.Join(dstDir, name)); err != nil {
			return err
		}
	}
	return nil
}

func sameDir(a, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return absA == absB, nil
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
