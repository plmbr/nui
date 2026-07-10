// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"sync"
)

type PiAgent struct {
	BinaryPath            string
	Sandbox               string
	DevcontainerWorkspace   string
	DevcontainerContainerID string

	sessionMu sync.Mutex
	session   *persistentPiSession
}

func (a *PiAgent) Name() string { return "pi" }

func (a *PiAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	if err := a.validateSandbox(); err != nil {
		return err
	}

	a.sessionMu.Lock()
	if a.session == nil {
		a.session = &persistentPiSession{}
	}
	sess := a.session
	a.sessionMu.Unlock()

	sessionID, err := sess.runTurn(ctx, a, req, events)
	if err != nil {
		return err
	}
	if sessionID == "" {
		sessionID = sess.currentSessionID()
	}
	events <- Event{Type: EventDone, SessionID: sessionID}
	return nil
}

func (a *PiAgent) validateSandbox() error {
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

func (a *PiAgent) Stop() {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	if a.session != nil {
		a.session.stop()
		a.session = nil
	}
}

func (a *PiAgent) binaryPath() string {
	if a.BinaryPath != "" {
		return a.BinaryPath
	}
	return "pi"
}

func (a *PiAgent) useBwrap() bool {
	return a.Sandbox == "bubblewrap" && GetBwrapStatus().Available
}

func (a *PiAgent) useDevcontainer() bool {
	return useDevcontainerSandbox(a.Sandbox, a.DevcontainerWorkspace)
}
