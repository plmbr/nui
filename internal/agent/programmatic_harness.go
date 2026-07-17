// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"nui/internal/extensions"
)

// programmaticHarnessAgent implements Agent using a shared programmatic extension host.
type programmaticHarnessAgent struct {
	inner     *extensions.ProgrammaticHarnessAgent
	agentID   string
	projectID string
}

func newProgrammaticHarnessAgent(ref extensions.HarnessRef, projectID string) *programmaticHarnessAgent {
	inner := ref.Extension.NewProgrammaticHarnessAgent(ref.AgentID, ref.Entry.ID, projectID)
	return &programmaticHarnessAgent{
		inner:     inner,
		agentID:   ref.AgentID,
		projectID: projectID,
	}
}

func (a *programmaticHarnessAgent) Name() string { return a.inner.Name() }

func (a *programmaticHarnessAgent) Stop() { a.inner.Stop() }

var programmaticRunID atomic.Int64

func (a *programmaticHarnessAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	runID := fmt.Sprintf("run-%d", programmaticRunID.Add(1))
	extra := programmaticHarnessExtras(req, a.projectID)
	return a.inner.RunHarness(ctx, req.Message, runID, extra, func(ev extensions.HarnessRunEvent) {
		if ev.Raw != nil {
			if e, ok := eventFromHarnessParams(ev.Raw); ok {
				events <- e
				return
			}
		}
		switch ev.Type {
		case "text":
			events <- Event{Type: EventText, Content: ev.Content}
		case "error":
			events <- Event{Type: EventError, Error: ev.Error}
		case "done":
			events <- Event{Type: EventDone}
		}
	})
}

func programmaticHarnessExtras(req RunRequest, projectID string) map[string]any {
	extra := map[string]any{}
	nuiSessionID := strings.TrimSpace(req.NuiSessionID)
	if nuiSessionID == "" {
		nuiSessionID = projectID
	}
	if nuiSessionID != "" {
		extra["nuiSessionId"] = nuiSessionID
	}
	if req.SessionID != "" {
		extra["sessionId"] = req.SessionID
	}
	if req.WorkingDir != "" {
		extra["workingDir"] = req.WorkingDir
	}
	if req.SystemPrompt != "" {
		extra["systemPrompt"] = req.SystemPrompt
	}
	if req.Model != "" {
		extra["model"] = req.Model
	}
	return extra
}
