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
	switch {
	case model.IsCouncilAgent(a.def):
		return a.runCouncil(ctx, req, events)
	case model.IsSubAgentsOrchestration(a.def):
		return a.runSubAgents(ctx, req, events)
	case model.IsMultiStepWorkflow(a.def):
		return a.runWorkflow(ctx, req, events)
	default:
		return a.runStep(ctx, req, a.def.Harness, nil, events)
	}
}

func (a *ADLAgent) runWorkflow(ctx context.Context, req RunRequest, events chan<- Event) error {
	steps := model.OrchestrationSteps(a.def)
	if len(steps) == 0 {
		return a.runStep(ctx, req, a.def.Harness, nil, events)
	}

	waves, err := topoWaves(steps)
	if err != nil {
		return err
	}

	stepOutputs := newStepOutputStore()
	stepIndex := 0
	totalSteps := len(steps)
	for _, wave := range waves {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if len(wave) == 1 {
			stepIndex++
			text, err := a.executeWorkflowStep(ctx, req, wave[0], stepIndex, totalSteps, stepOutputs, events)
			if err != nil {
				return err
			}
			stepOutputs.set(wave[0], text)
			continue
		}

		type stepResult struct {
			step model.ADLStep
			text string
			err  error
		}
		results := make([]stepResult, len(wave))
		var wg sync.WaitGroup
		for i, step := range wave {
			wg.Add(1)
			go func(i int, step model.ADLStep) {
				defer wg.Done()
				sink := &collectingEvents{}
				ch := sink.start()
				text, err := a.executeWorkflowStep(ctx, req, step, stepIndex+i+1, totalSteps, stepOutputs, ch)
				sink.finish()
				if strings.TrimSpace(text) == "" {
					text = sink.text
				}
				results[i] = stepResult{step: step, text: text, err: err}
			}(i, step)
		}
		wg.Wait()
		stepIndex += len(wave)
		for _, r := range results {
			if r.err != nil {
				return r.err
			}
			stepOutputs.set(r.step, r.text)
			if totalSteps > 1 {
				events <- Event{Type: EventText, Content: fmt.Sprintf("\n**Step: %s** (parallel)\n\n%s\n", r.step.Name, r.text)}
			}
		}
	}
	return nil
}

func (a *ADLAgent) executeWorkflowStep(
	ctx context.Context,
	req RunRequest,
	step model.ADLStep,
	stepIndex, totalSteps int,
	stepOutputs stepOutputStore,
	events chan<- Event,
) (string, error) {
	if totalSteps > 1 {
		events <- Event{Type: EventText, Content: fmt.Sprintf("\n**Step %d/%d: %s**\n\n", stepIndex, totalSteps, step.Name)}
	}
	if step.Type == "hitl" {
		return a.runHITLStep(ctx, req, step, stepOutputs, events)
	}

	harness, systemPrompt, stepDef, err := a.resolveStepExecution(req, step)
	if err != nil {
		return "", err
	}
	msg := buildStepMessage(req.Message, step, stepOutputs)
	stepReq := RunRequest{
		NuiSessionID:     a.projectID,
		RunID:            req.RunID,
		WorkingDir:       req.WorkingDir,
		Message:          msg,
		SystemPrompt:     systemPrompt,
		UserScopeHarness: req.UserScopeHarness,
		ResolveADL:       req.ResolveADL,
		AgentConfig:      req.AgentConfig,
	}
	collecting := &collectingEvents{upstream: events}
	stepEvents := collecting.start()
	runAgent := a
	if stepDef != nil {
		runAgent = NewADLAgent(*stepDef, a.projectID, a.manager)
	}
	if err := runAgent.runStep(ctx, stepReq, harness, &step, stepEvents); err != nil {
		collecting.finish()
		return collecting.text, err
	}
	collecting.finish()
	return collecting.text, nil
}

func (a *ADLAgent) resolveStepExecution(req RunRequest, step model.ADLStep) (model.ADLHarness, string, *model.ADLDefinition, error) {
	harness := a.def.Harness
	systemPrompt := a.def.SystemPrompt
	if step.Harness != nil {
		harness = *step.Harness
	}
	if step.SystemPrompt != "" {
		systemPrompt = step.SystemPrompt
	}
	agentID := strings.TrimSpace(step.Agent)
	if agentID == "" {
		return harness, systemPrompt, nil, nil
	}
	if req.ResolveADL == nil {
		return harness, systemPrompt, nil, fmt.Errorf("step %q references agent %q but ResolveADL is not configured", step.Name, agentID)
	}
	def, ok := req.ResolveADL(agentID)
	if !ok {
		return harness, systemPrompt, nil, fmt.Errorf("step %q: unknown agent %q", step.Name, agentID)
	}
	harness = def.Harness
	if step.Harness != nil {
		harness = *step.Harness
	}
	systemPrompt = def.SystemPrompt
	if step.SystemPrompt != "" {
		systemPrompt = step.SystemPrompt
	}
	return harness, systemPrompt, &def, nil
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
	deps.Sandbox = harness.Sandbox
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
		req.Env = mergenuiHarnessEnv(mergeADLEnv(a.def, harness), nuiSessionIDForRun(req, a.projectID), req.RunID, defaultnuiAPIURL())
	}
	if req.HarnessPermissions == "" {
		req.HarnessPermissions = hitl.EffectivePermissions(a.def, req.AgentConfig)
	}
	if req.ToolApprovalPolicy == "" {
		req.ToolApprovalPolicy, req.ToolApprovalTools = hitl.EffectiveToolApprovals(a.def, req.AgentConfig)
	}
	if harness.Type == "api" {
		if len(req.MCPServers) == 0 {
			req.MCPServers = deps.MCPServers
		}
		req.APIProvider = harness.Provider
		req.Model = resolveAPIModel(req, harness)
	} else if harness.Type == "antigravity" {
		req.Model = resolveAntigravityModel(req, harness)
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
	waves, err := topoWaves(steps)
	if err != nil {
		return nil, err
	}
	var sorted []model.ADLStep
	for _, wave := range waves {
		sorted = append(sorted, wave...)
	}
	return sorted, nil
}

// topoWaves groups steps into waves where each wave's members have no unmet dependencies
// and may run concurrently.
func topoWaves(steps []model.ADLStep) ([][]model.ADLStep, error) {
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

	remaining := make([]bool, len(steps))
	for i := range remaining {
		remaining[i] = true
	}

	var waves [][]model.ADLStep
	placed := 0
	for placed < len(steps) {
		var wave []model.ADLStep
		var waveIdx []int
		for i, d := range inDegree {
			if remaining[i] && d == 0 {
				wave = append(wave, steps[i])
				waveIdx = append(waveIdx, i)
			}
		}
		if len(wave) == 0 {
			return nil, fmt.Errorf("ADL steps contain a cycle")
		}
		for _, i := range waveIdx {
			remaining[i] = false
			placed++
			for j, s := range steps {
				if !remaining[j] {
					continue
				}
				for _, dep := range s.DependsOn {
					if dep == steps[i].Name {
						inDegree[j]--
					}
				}
			}
		}
		waves = append(waves, wave)
	}
	return waves, nil
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
			if ev.Type != EventDone && c.upstream != nil {
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
