// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

var codexBinaryPaths = []string{
	"codex",
	"/Applications/Codex.app/Contents/Resources/codex",
}

func findCodexBinary() string {
	if p := os.Getenv("NUI_CODEX_PATH"); p != "" {
		return p
	}
	for _, p := range codexBinaryPaths {
		if path, err := exec.LookPath(p); err == nil {
			return path
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return codexBinaryPaths[0]
}

// CLIAvailable reports whether the CLI required for harnessType is installed.
func CLIAvailable(harnessType string) bool {
	switch harnessType {
	case "claude-code":
		_, err := exec.LookPath("claude")
		return err == nil
	case "codex":
		if p := os.Getenv("NUI_CODEX_PATH"); p != "" {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return true
			}
		}
		for _, p := range codexBinaryPaths {
			if _, err := exec.LookPath(p); err == nil {
				return true
			}
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
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

type CodexAgent struct {
	BinaryPath            string
	Sandbox               string
	DevcontainerWorkspace   string
	DevcontainerContainerID string

	sessionMu sync.Mutex
	session   *persistentCodexSession
}

func (a *CodexAgent) Name() string { return "codex" }

func (a *CodexAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	if err := a.validateSandbox(); err != nil {
		return err
	}

	a.sessionMu.Lock()
	if a.session == nil || !a.session.matches(a, req) {
		if a.session != nil {
			a.session.stop()
		}
		a.session = &persistentCodexSession{}
	}
	sess := a.session
	a.sessionMu.Unlock()

	return sess.runTurn(ctx, a, req, events)
}

func (a *CodexAgent) validateSandbox() error {
	if err := requireDevcontainer(a.Sandbox, a.DevcontainerWorkspace); err != nil {
		return err
	}
	if a.Sandbox != "bubblewrap" {
		return nil
	}
	bwrap := GetBwrapStatus()
	if !bwrap.Available {
		return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
	}
	return nil
}

func (a *CodexAgent) Stop() {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	if a.session != nil {
		a.session.stop()
		a.session = nil
	}
}

func (a *CodexAgent) binaryPath() string {
	if a.BinaryPath != "" {
		return a.BinaryPath
	}
	return findCodexBinary()
}

func (a *CodexAgent) useBwrap() bool {
	switch a.Sandbox {
	case "bubblewrap":
		return GetBwrapStatus().Available
	case "none":
		return false
	default:
		return false
	}
}

func (a *CodexAgent) useDevcontainer() bool {
	return useDevcontainerSandbox(a.Sandbox, a.DevcontainerWorkspace)
}
