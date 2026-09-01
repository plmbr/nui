// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

var antigravityBinaryPaths = []string{
	"agy",
}

func findAntigravityBinary() string {
	if p := os.Getenv("NUI_ANTIGRAVITY_PATH"); p != "" {
		return p
	}
	for _, p := range antigravityBinaryPaths {
		if path, err := exec.LookPath(p); err == nil {
			return path
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return antigravityBinaryPaths[0]
}

func antigravityCLIAvailable() bool {
	if p := os.Getenv("NUI_ANTIGRAVITY_PATH"); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return true
		}
	}
	for _, p := range antigravityBinaryPaths {
		if _, err := exec.LookPath(p); err == nil {
			return true
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

type AntigravityAgent struct {
	BinaryPath              string
	Sandbox                 string
	DevcontainerWorkspace   string
	DevcontainerContainerID string

	sessionMu sync.Mutex
	session   *persistentAntigravitySession
}

func (a *AntigravityAgent) Name() string { return "antigravity" }

func (a *AntigravityAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	if err := a.validateSandbox(); err != nil {
		return err
	}

	a.sessionMu.Lock()
	if a.session == nil || !a.session.matches(a, req) {
		if a.session != nil {
			a.session.stop()
		}
		a.session = &persistentAntigravitySession{}
	}
	sess := a.session
	a.sessionMu.Unlock()

	return sess.runTurn(ctx, a, req, events)
}

func (a *AntigravityAgent) validateSandbox() error {
	if a.Sandbox == "docker" {
		return fmt.Errorf("antigravity docker sandbox is not supported yet")
	}
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

func (a *AntigravityAgent) Stop() {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	if a.session != nil {
		a.session.stop()
		a.session = nil
	}
}

func (a *AntigravityAgent) binaryPath() string {
	if a.BinaryPath != "" {
		return a.BinaryPath
	}
	return findAntigravityBinary()
}

func (a *AntigravityAgent) useBwrap() bool {
	switch a.Sandbox {
	case "bubblewrap":
		return GetBwrapStatus().Available
	default:
		return false
	}
}

func (a *AntigravityAgent) useDevcontainer() bool {
	return useDevcontainerSandbox(a.Sandbox, a.DevcontainerWorkspace)
}
