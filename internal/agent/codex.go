// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "os/exec"

// codexBinaryPaths lists locations to search for the codex binary in order.
var codexBinaryPaths = []string{
	"codex",
	"/Applications/Codex.app/Contents/Resources/codex",
}

// CLIAvailable reports whether the CLI required for harnessType is installed.
func CLIAvailable(harnessType string) bool {
	switch harnessType {
	case "claude-code":
		_, err := exec.LookPath("claude")
		return err == nil
	case "codex":
		for _, p := range codexBinaryPaths {
			if _, err := exec.LookPath(p); err == nil {
				return true
			}
		}
		return false
	case "pi":
		_, err := exec.LookPath("pi")
		return err == nil
	case "opencode":
		_, err := exec.LookPath("opencode")
		return err == nil
	default:
		return true
	}
}
