// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ensureDesktopPATH merges the user's login-shell PATH (and common install
// locations) into the process environment. macOS GUI apps launched from
// Finder/Dock inherit a stripped PATH (/usr/bin:/bin:…), so LookPath cannot
// find agent CLIs like claude/pi/codex/opencode that live under Homebrew,
// ~/.local/bin, nvm, etc. Unavailable CLIs are hidden from the new-session UI.
func ensureDesktopPATH() {
	current := os.Getenv("PATH")
	var parts []string
	if login, err := loginShellPATH(2 * time.Second); err == nil && login != "" {
		parts = append(parts, splitPATH(login)...)
	}
	parts = append(parts, commonUserPathDirs()...)
	parts = append(parts, splitPATH(current)...)
	merged := joinPATH(dedupePATH(parts))
	if merged == "" || merged == current {
		return
	}
	_ = os.Setenv("PATH", merged)
}

func splitPATH(path string) []string {
	if path == "" {
		return nil
	}
	return filepath.SplitList(path)
}

func joinPATH(parts []string) string {
	return strings.Join(parts, string(os.PathListSeparator))
}

func dedupePATH(parts []string) []string {
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := p
		if runtime.GOOS == "windows" {
			key = strings.ToLower(p)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

func commonUserPathDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	candidates := []string{
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/bin",
		"/usr/local/sbin",
	}
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, "bin"),
			filepath.Join(home, ".opencode", "bin"),
			filepath.Join(home, ".bun", "bin"),
			filepath.Join(home, ".cargo", "bin"),
		)
		// Active nvm default, if present (avoid scanning all versions).
		nvmDefault := filepath.Join(home, ".nvm", "alias", "default")
		if data, err := os.ReadFile(nvmDefault); err == nil {
			ver := strings.TrimSpace(string(data))
			if ver != "" && !strings.Contains(ver, "..") && !strings.ContainsAny(ver, `/\\`) {
				nodeRoot := filepath.Join(home, ".nvm", "versions", "node")
				for _, name := range []string{ver, "v" + strings.TrimPrefix(ver, "v")} {
					candidates = append(candidates, filepath.Join(nodeRoot, name, "bin"))
				}
			}
		}
	}
	out := make([]string, 0, len(candidates))
	for _, dir := range candidates {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			out = append(out, dir)
		}
	}
	return out
}

func loginShellPATH(timeout time.Duration) (string, error) {
	shell := strings.TrimSpace(os.Getenv("SHELL"))
	if shell == "" {
		if runtime.GOOS == "windows" {
			return "", fmt.Errorf("no SHELL")
		}
		shell = "/bin/zsh"
		if _, err := os.Stat(shell); err != nil {
			shell = "/bin/bash"
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// Login shell only (-l). Avoid -i: interactive shells can hang without a TTY.
	cmd := exec.CommandContext(ctx, shell, "-lc", `printf %s "$PATH"`)
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("login shell PATH: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
