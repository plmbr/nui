// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"strings"

	"nui/internal/hitl"
	"nui/internal/model"
)

type resolvedSubAgent struct {
	id    string
	def   model.ADLDefinition
	label string
}

func (a *ADLAgent) runOrchestrator(ctx context.Context, req RunRequest, events chan<- Event) error {
	resolve := req.ResolveADL
	if resolve == nil {
		return fmt.Errorf("orchestrator requires ResolveADL")
	}

	candidates, err := a.resolveSubAgents(resolve)
	if err != nil {
		return err
	}

	selected, err := a.routeToSubAgent(ctx, req, candidates)
	if err != nil {
		return err
	}

	events <- Event{
		Type:             EventSubAgentRouted,
		RoutedAgentID:    selected.id,
		RoutedAgentLabel: selected.label,
	}

	subAgent := NewADLAgent(selected.def, a.projectID, a.manager)
	subReq := req
	subReq.Message = req.Message
	subReq.SystemPrompt = ""
	subReq.SessionID = ""
	if req.SubAgentHarnessSession != nil {
		subReq.SessionID = req.SubAgentHarnessSession(selected.id)
	}
	subReq.HarnessPermissions = hitl.EffectivePermissions(selected.def, req.AgentConfig)
	subReq.ToolApprovalPolicy, subReq.ToolApprovalTools = hitl.EffectiveToolApprovals(selected.def, req.AgentConfig)

	collecting := &subAgentEventCollector{
		upstream: events,
		onDone: func(harnessSessionID string) {
			if harnessSessionID != "" && req.OnSubAgentHarnessSession != nil {
				req.OnSubAgentHarnessSession(selected.id, harnessSessionID)
			}
		},
	}
	subEvents := collecting.start()
	defer collecting.finish()

	return subAgent.Run(ctx, subReq, subEvents)
}

func (a *ADLAgent) resolveSubAgents(resolve ADLResolver) ([]resolvedSubAgent, error) {
	var out []resolvedSubAgent
	for _, id := range a.def.SubAgents {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		def, ok := resolve(id)
		if !ok {
			return nil, fmt.Errorf("subAgents: unknown agent %q", id)
		}
		out = append(out, resolvedSubAgent{
			id:    model.ADLAgentID(def),
			def:   def,
			label: model.ADLAgentLabel(def),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("orchestrator has no valid sub-agents")
	}
	return out, nil
}

func (a *ADLAgent) routeToSubAgent(ctx context.Context, req RunRequest, candidates []resolvedSubAgent) (resolvedSubAgent, error) {
	routingPrompt := buildRoutingPrompt(req.Message, candidates)
	routeReq := req
	routeReq.Message = routingPrompt
	if routeReq.SystemPrompt == "" {
		routeReq.SystemPrompt = defaultRoutingSystemPrompt()
	}

	collector := &textCollector{}
	collector.ch = make(chan Event, 64)
	done := make(chan struct{})
	go collector.drain(done)

	if err := a.RunEphemeral(ctx, routeReq, collector.ch); err != nil {
		close(collector.ch)
		<-done
		return candidates[0], nil
	}
	close(collector.ch)
	<-done

	if picked, ok := parseRoutingResponse(collector.text, candidates); ok {
		return picked, nil
	}
	return candidates[0], nil
}

func defaultRoutingSystemPrompt() string {
	return `You are a router. Given the user message and the list of available sub-agents, respond with ONLY the agent id of the best match. No explanation.`
}

func buildRoutingPrompt(userMessage string, candidates []resolvedSubAgent) string {
	var b strings.Builder
	b.WriteString("User message:\n")
	b.WriteString(userMessage)
	b.WriteString("\n\nAvailable sub-agents:\n")
	for _, c := range candidates {
		fmt.Fprintf(&b, "- id: %s\n  name: %s\n", c.id, c.label)
		if desc := strings.TrimSpace(c.def.Description); desc != "" {
			fmt.Fprintf(&b, "  description: %s\n", desc)
		}
	}
	b.WriteString("\nRespond with only the agent id of the best match.")
	return b.String()
}

func parseRoutingResponse(text string, candidates []resolvedSubAgent) (resolvedSubAgent, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return resolvedSubAgent{}, false
	}
	firstLine := strings.TrimSpace(strings.Split(text, "\n")[0])
	firstLine = strings.Trim(firstLine, `"'.,;:`)
	for _, c := range candidates {
		if strings.EqualFold(c.id, firstLine) || strings.EqualFold(c.label, firstLine) {
			return c, true
		}
	}
	first := strings.Fields(firstLine)
	if len(first) == 0 {
		return resolvedSubAgent{}, false
	}
	firstToken := strings.Trim(first[0], `"'.,;:`)
	lowerText := strings.ToLower(text)
	for _, c := range candidates {
		if strings.EqualFold(c.id, firstToken) || strings.EqualFold(c.label, firstToken) {
			return c, true
		}
		if strings.Contains(lowerText, strings.ToLower(c.id)) {
			return c, true
		}
	}
	return resolvedSubAgent{}, false
}

type textCollector struct {
	text string
	ch   chan Event
}

func (c *textCollector) drain(done chan struct{}) {
	defer close(done)
	for ev := range c.ch {
		if ev.Type == EventText {
			c.text += ev.Content
		}
	}
}

type subAgentEventCollector struct {
	upstream chan<- Event
	onDone   func(harnessSessionID string)
	pipe     chan Event
	done     chan struct{}
}

func (c *subAgentEventCollector) start() chan<- Event {
	c.pipe = make(chan Event, 64)
	c.done = make(chan struct{})
	go func() {
		defer close(c.done)
		for ev := range c.pipe {
			if ev.Type == EventDone {
				if c.onDone != nil {
					c.onDone(ev.SessionID)
				}
				c.upstream <- ev
				continue
			}
			c.upstream <- ev
		}
	}()
	return c.pipe
}

func (c *subAgentEventCollector) finish() {
	if c.pipe == nil {
		return
	}
	close(c.pipe)
	<-c.done
	c.pipe = nil
}
