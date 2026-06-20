// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

// codexBinaryPaths lists locations to search for the codex binary in order.
var codexBinaryPaths = []string{
	"codex",
	"/Applications/Codex.app/Contents/Resources/codex",
}

func findCodexBinary() string {
	for _, p := range codexBinaryPaths {
		if path, err := exec.LookPath(p); err == nil {
			return path
		}
	}
	return codexBinaryPaths[0] // let exec fail with a clear error
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

// CodexAgent runs the Codex CLI non-interactively and streams events back.
type CodexAgent struct {
	// BinaryPath overrides the codex binary location; auto-detected if empty.
	BinaryPath string
	// Model overrides the model (e.g. "o3").
	Model string
	// Sandbox controls sandboxing: "none" disables bwrap, "bubblewrap" forces it, "" uses none.
	Sandbox string

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

func (a *CodexAgent) modelName() string {
	return a.Model
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
