// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "context"

// HarnessRunHook replaces real harness execution when set (used by server integration tests).
type HarnessRunHook func(ctx context.Context, req RunRequest, events chan<- Event) error

// SetTestHarnessRun installs a hook that ADLAgent.dispatchHarness invokes before real harnesses.
func (m *Manager) SetTestHarnessRun(hook HarnessRunHook) {
	m.testHarnessRun = hook
}

// harnessRunAgent delegates Run to a HarnessRunHook.
type harnessRunAgent struct {
	name string
	hook HarnessRunHook
}

func (a *harnessRunAgent) Name() string { return a.name }

func (a *harnessRunAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	if a.hook == nil {
		events <- Event{Type: EventText, Content: "test stub"}
		events <- Event{Type: EventDone, SessionID: "test-session"}
		return nil
	}
	return a.hook(ctx, req, events)
}
