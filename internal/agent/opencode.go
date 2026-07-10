// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"sync"
)

type OpenCodeAgent struct {
	BinaryPath            string
	Sandbox               string
	DevcontainerWorkspace   string
	DevcontainerContainerID string

	sessionMu sync.Mutex
	session   *persistentOpenCodeSession
}

func (a *OpenCodeAgent) Name() string { return "opencode" }

func (a *OpenCodeAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	if err := a.validateSandbox(); err != nil {
		return err
	}

	a.sessionMu.Lock()
	if a.session == nil {
		a.session = &persistentOpenCodeSession{}
	}
	sess := a.session
	a.sessionMu.Unlock()

	sessionID, err := sess.runTurn(ctx, a, req, events)
	if err != nil {
		return err
	}
	events <- Event{Type: EventDone, SessionID: sessionID}
	return nil
}

func (a *OpenCodeAgent) validateSandbox() error {
	if useDevcontainerSandbox(a.Sandbox, a.DevcontainerWorkspace) {
		if err := requireDevcontainer(a.Sandbox, a.DevcontainerWorkspace); err != nil {
			return err
		}
		if a.DevcontainerContainerID == "" {
			return fmt.Errorf("opencode devcontainer requires a running container")
		}
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

func (a *OpenCodeAgent) Stop() {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	if a.session != nil {
		a.session.stop()
		a.session = nil
	}
}

func (a *OpenCodeAgent) binaryPath() string {
	if a.BinaryPath != "" {
		return a.BinaryPath
	}
	return "opencode"
}

func (a *OpenCodeAgent) useBwrap() bool {
	return a.Sandbox == "bubblewrap" && GetBwrapStatus().Available
}

func (a *OpenCodeAgent) useDevcontainer() bool {
	return useDevcontainerSandbox(a.Sandbox, a.DevcontainerWorkspace)
}
