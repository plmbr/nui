// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"nui/internal/extensions"
	"nui/internal/hitl"
	"nui/internal/mcpoauth"
	"nui/internal/model"
)

// ADLAgent executes an ADLDefinition by running its steps in topological order.
type ADLAgent struct {
	def       model.ADLDefinition
	projectID string
	manager   *Manager
}

func NewADLAgent(def model.ADLDefinition, projectID string, manager *Manager) *ADLAgent {
	return &ADLAgent{def: def, projectID: projectID, manager: manager}
}

func (a *ADLAgent) Name() string { return "adl:" + model.ADLAgentID(a.def) }

// RunEphemeral executes a single top-level harness turn without resuming the session's
// main agent conversation or polluting its persistent harness cache entry.
func (a *ADLAgent) RunEphemeral(ctx context.Context, req RunRequest, events chan<- Event) error {
	req.Ephemeral = true
	req.SessionID = ""
	req.HarnessPermissions = hitl.PermissionsBypass
	return a.runStep(ctx, req, a.def.Harness, nil, events)
}

func (a *ADLAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	if model.IsOrchestratorAgent(a.def) {
		return a.runOrchestrator(ctx, req, events)
	}

	steps := a.def.Steps
	if len(steps) == 0 {
		// Single-step definition — run the top-level harness directly.
		return a.runStep(ctx, req, a.def.Harness, nil, events)
	}

	sorted, err := topoSort(steps)
	if err != nil {
		return err
	}

	// stepOutputs holds accumulated text output from each step (named outputs supported).
	stepOutputs := newStepOutputStore()

	for i, step := range sorted {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Emit step header for multi-step pipelines.
		if len(sorted) > 1 {
			events <- Event{Type: EventText, Content: fmt.Sprintf("\n**Step %d/%d: %s**\n\n", i+1, len(sorted), step.Name)}
		}

		if step.Type == "hitl" {
			out, err := a.runHITLStep(ctx, req, step, stepOutputs, events)
			if err != nil {
				return err
			}
			stepOutputs.setRaw(step.Name, out)
			continue
		}

		harness := a.def.Harness
		if step.Harness != nil {
			harness = *step.Harness
		}
		systemPrompt := a.def.SystemPrompt
		if step.SystemPrompt != "" {
			systemPrompt = step.SystemPrompt
		}

		msg := buildStepMessage(req.Message, step, stepOutputs)

		stepReq := RunRequest{
			NuiSessionID:    a.projectID,
			RunID:            req.RunID,
			WorkingDir:       req.WorkingDir,
			Message:          msg,
			SystemPrompt:     systemPrompt,
			UserScopeHarness: req.UserScopeHarness,
		}

		// Collect this step's output so downstream steps can reference it.
		collecting := &collectingEvents{upstream: events}
		stepEvents := collecting.start()
		if err := a.runStep(ctx, stepReq, harness, &step, stepEvents); err != nil {
			collecting.finish()
			return err
		}
		collecting.finish()
		stepOutputs.set(step, collecting.text)
	}

	return nil
}

// runStep resolves the harness and runs the agent for a single step.
func (a *ADLAgent) runStep(ctx context.Context, req RunRequest, harness model.ADLHarness, step *model.ADLStep, events chan<- Event) error {
	req.UserScopeHarness = effectiveUserScopeHarness(harness.Type, req.UserScopeHarness)
	titleSystemPrompt := req.SystemPrompt
	var reg *extensions.Registry
	if a.manager != nil {
		reg = a.manager.registry
	}
	var deps HarnessDeps
	var err error
	if req.Ephemeral {
		deps = HarnessDeps{WorkingDir: req.WorkingDir}
	} else {
		deps, err = buildHarnessDeps(a.projectID, a.def, step, req.WorkingDir, reg, req.AgentConfig)
		if err != nil {
			return fmt.Errorf("expand harness deps: %w", err)
		}
	}
	deps.UserScope = req.UserScopeHarness
	configDir, err := ProvisionHarnessConfig(a.projectID, harness.Type, deps)
	if err != nil {
		return fmt.Errorf("provision harness config: %w", err)
	}
	req.ConfigDir = configDir
	if titleSystemPrompt != "" {
		req.SystemPrompt = titleSystemPrompt
	} else {
		req.SystemPrompt = deps.SystemPrompt
	}
	req.Model = harness.Model
	if req.Ephemeral {
		req.Env = mergeADLEnv(a.def, harness)
	} else {
		req.Env = mergenuiHarnessEnv(mergeADLEnv(a.def, harness), loopSessionIDForRun(req, a.projectID), req.RunID, defaultnuiAPIURL())
	}
	if req.HarnessPermissions == "" {
		req.HarnessPermissions = hitl.EffectivePermissions(a.def, req.AgentConfig)
	}
	if req.ToolApprovalPolicy == "" {
		req.ToolApprovalPolicy, req.ToolApprovalTools = hitl.EffectiveToolApprovals(a.def, req.AgentConfig)
	}
	if harness.Type == "api" {
		req.MCPServers = deps.MCPServers
		req.APIProvider = harness.Provider
		req.Model = resolveAPIModel(req, harness)
	} else if !req.Ephemeral && len(deps.MCPServers) > 0 {
		for _, msg := range mcpoauth.ProbeConnectFailures(ctx, deps.MCPServers) {
			events <- Event{Type: EventText, Content: msg + "\n"}
		}
	}

	return a.dispatchHarness(ctx, req, harness, events)
}

// buildStepMessage is defined in adl_step_io.go.

// topoSort returns steps in dependency order using Kahn's algorithm.
func topoSort(steps []model.ADLStep) ([]model.ADLStep, error) {
	index := map[string]int{}
	for i, s := range steps {
		index[s.Name] = i
	}

	inDegree := make([]int, len(steps))
	for _, s := range steps {
		for _, dep := range s.DependsOn {
			if _, ok := index[dep]; !ok {
				return nil, fmt.Errorf("step %q depends on unknown step %q", s.Name, dep)
			}
			inDegree[index[s.Name]]++
		}
	}

	queue := []int{}
	for i, d := range inDegree {
		if d == 0 {
			queue = append(queue, i)
		}
	}

	var sorted []model.ADLStep
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		sorted = append(sorted, steps[cur])
		// Find steps that depend on this one.
		for i, s := range steps {
			for _, dep := range s.DependsOn {
				if dep == steps[cur].Name {
					inDegree[i]--
					if inDegree[i] == 0 {
						queue = append(queue, i)
					}
				}
			}
		}
	}

	if len(sorted) != len(steps) {
		return nil, fmt.Errorf("ADL steps contain a cycle")
	}
	return sorted, nil
}

// collectingEvents pipes events through to an upstream channel while capturing text output.
type collectingEvents struct {
	upstream chan<- Event
	text     string
	pipe     chan Event
	wg       sync.WaitGroup
}

func (c *collectingEvents) start() chan<- Event {
	c.pipe = make(chan Event, 64)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for ev := range c.pipe {
			if ev.Type == EventText {
				c.text += ev.Content
			}
			// Forward all events except EventDone from intermediate steps
			// so the caller controls when to emit the final done.
			if ev.Type != EventDone {
				c.upstream <- ev
			}
		}
	}()
	return c.pipe
}

func (c *collectingEvents) finish() {
	if c.pipe == nil {
		return
	}
	close(c.pipe)
	c.wg.Wait()
	c.pipe = nil
}

func (a *ADLAgent) runHITLStep(ctx context.Context, req RunRequest, step model.ADLStep, outputs stepOutputStore, events chan<- Event) (string, error) {
	gate := orchestrationGateFn()
	if gate == nil {
		return "", fmt.Errorf("orchestration HITL gate not configured")
	}
	if step.HITL == nil {
		return "", fmt.Errorf("step %q: hitl block required for type hitl", step.Name)
	}
	kind := step.HITL.Kind
	if kind == "" {
		kind = "approval"
	}
	payload := map[string]any{
		"title":   step.HITL.Title,
		"message": step.HITL.Message,
	}
	if len(step.HITL.Questions) > 0 {
		payload["questions"] = step.HITL.Questions
	}
	if len(step.HITL.Actions) > 0 {
		actions := make([]map[string]string, 0, len(step.HITL.Actions))
		for _, act := range step.HITL.Actions {
			actions = append(actions, map[string]string{"id": act.ID, "label": act.Label})
		}
		payload["actions"] = actions
	}
	for _, disp := range step.HITL.Display {
		if text, ok := outputs.resolve(disp.From); ok {
			key := "display_" + strings.ReplaceAll(disp.From, ".", "_")
			payload[key] = text
		}
	}
	channels := []string{"nui-ui"}
	if len(step.HITL.Channels) > 0 {
		channels = append([]string{}, step.HITL.Channels...)
	}
	created, err := gate.CreateOrchestrationGate(ctx, hitl.CreateInput{
		SessionID: a.projectID,
		RunID:     req.RunID,
		StepName:  step.Name,
		Kind:      kind,
		Routing:   hitl.Routing{Channels: channels},
		Payload:   payload,
	})
	if err != nil {
		return "", err
	}
	events <- Event{
		Type:    EventHITLRequest,
		Content: created.RequestID,
	}
	resp, err := gate.Wait(ctx, created.RequestID)
	if err != nil {
		return "", err
	}
	if resp.Status == hitl.StatusDeclined || resp.Status == hitl.StatusCancelled {
		return "", fmt.Errorf("hitl gate %q %s", step.Name, resp.Status)
	}
	if action, ok := resp.Answers["action"].(string); ok && action != "" {
		return action, nil
	}
	if answer, ok := resp.Answers["answer"].(string); ok {
		return answer, nil
	}
	data, _ := json.Marshal(resp.Answers)
	return string(data), nil
}
