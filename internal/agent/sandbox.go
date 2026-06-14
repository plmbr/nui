// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BwrapStatus holds the result of bubblewrap availability detection.
type BwrapStatus struct {
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
	Error     string `json:"error,omitempty"`
}

var (
	bwrapOnce   sync.Once
	bwrapStatus BwrapStatus
)

// GetBwrapStatus returns the cached bwrap detection result, running detection on first call.
func GetBwrapStatus() BwrapStatus {
	bwrapOnce.Do(func() {
		bwrapStatus = detectBwrap()
	})
	return bwrapStatus
}

func detectBwrap() BwrapStatus {
	path, err := exec.LookPath("bwrap")
	if err != nil {
		return BwrapStatus{Available: false, Error: "bwrap not found in PATH"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx,
		path,
		"--unshare-user",
		"--ro-bind", "/", "/",
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		"--",
		"echo", "ok",
	).Output()
	if err != nil {
		return BwrapStatus{Available: false, Path: path, Error: fmt.Sprintf("bwrap smoke test failed: %v", err)}
	}
	if strings.TrimSpace(string(out)) != "ok" {
		return BwrapStatus{Available: false, Path: path, Error: "bwrap smoke test: unexpected output"}
	}

	return BwrapStatus{Available: true, Path: path}
}

// WrapWithBwrap returns the bwrap binary and args that sandbox bin+args under workDir.
// workDir and ~/.claude are bind-mounted read-write; everything else is read-only.
// Network access is preserved so the claude CLI can reach Anthropic's API.
func WrapWithBwrap(bwrapPath, bin string, args []string, workDir string) (string, []string) {
	if workDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workDir = wd
		}
	}

	home, _ := os.UserHomeDir()
	claudeDir := filepath.Join(home, ".claude")
	os.MkdirAll(claudeDir, 0700) //nolint:errcheck

	bwrapArgs := []string{
		"--unshare-user",
		"--unshare-ipc",
		"--unshare-pid",
		"--unshare-uts",
		"--ro-bind", "/", "/",
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		"--die-with-parent",
		"--bind", claudeDir, claudeDir,
	}

	if workDir != "" {
		bwrapArgs = append(bwrapArgs, "--bind", workDir, workDir)
		bwrapArgs = append(bwrapArgs, "--chdir", workDir)
	}

	bwrapArgs = append(bwrapArgs, "--")
	bwrapArgs = append(bwrapArgs, bin)
	bwrapArgs = append(bwrapArgs, args...)

	return bwrapPath, bwrapArgs
}
