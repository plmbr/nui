// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"sync"
)

type ClaudeCodeAgent struct {
	// BinaryPath overrides the claude binary location; defaults to "claude" on PATH.
	BinaryPath string
	// Sandbox controls sandboxing: "none" (default), "bubblewrap", or "devcontainer".
	Sandbox string
	// DevcontainerWorkspace is the Loop-managed folder for devcontainer up/exec.
	DevcontainerWorkspace string
	DevcontainerContainerID string

	sessionMu sync.Mutex
	session   *persistentClaudeSession
}

func (a *ClaudeCodeAgent) Name() string { return "claude-code" }

func (a *ClaudeCodeAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	if err := a.validateSandbox(); err != nil {
		return err
	}

	a.sessionMu.Lock()
	if a.session == nil {
		a.session = &persistentClaudeSession{}
	}
	sess := a.session
	a.sessionMu.Unlock()

	return sess.runTurn(ctx, a, req, events)
}

func (a *ClaudeCodeAgent) validateSandbox() error {
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

func (a *ClaudeCodeAgent) Stop() {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	if a.session != nil {
		a.session.stop()
		a.session = nil
	}
}

func (a *ClaudeCodeAgent) binaryPath() string {
	if a.BinaryPath != "" {
		return a.BinaryPath
	}
	return "claude"
}

func (a *ClaudeCodeAgent) useBwrap() bool {
	return a.Sandbox == "bubblewrap" && GetBwrapStatus().Available
}

func (a *ClaudeCodeAgent) useDevcontainer() bool {
	return useDevcontainerSandbox(a.Sandbox, a.DevcontainerWorkspace)
}
