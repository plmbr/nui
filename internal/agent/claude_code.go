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
	// Model overrides the default model (e.g. "claude-opus-4-8").
	Model string
	// Sandbox controls sandboxing: "none" disables bwrap, "bubblewrap" forces it,
	// "" auto-detects (uses bwrap if available — legacy behaviour).
	Sandbox string

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

func (a *ClaudeCodeAgent) modelName() string {
	if a.Model != "" {
		return a.Model
	}
	return "claude-sonnet-4-6"
}

func (a *ClaudeCodeAgent) useBwrap() bool {
	switch a.Sandbox {
	case "bubblewrap":
		return GetBwrapStatus().Available
	case "none":
		return false
	default:
		return GetBwrapStatus().Available
	}
}
