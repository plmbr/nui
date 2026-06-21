// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"strings"

	"loop/internal/model"
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

func (a *ADLAgent) Name() string { return "adl:" + a.def.Name }

func (a *ADLAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	steps := a.def.Steps
	if len(steps) == 0 {
		// Single-step definition — run the top-level harness directly.
		return a.runStep(ctx, req, a.def.Harness, a.def.SystemPrompt, nil, events)
	}

	sorted, err := topoSort(steps)
	if err != nil {
		return err
	}

	// stepOutputs holds the accumulated text output from each named step.
	stepOutputs := map[string]string{}

	for i, step := range sorted {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Emit step header for multi-step pipelines.
		if len(sorted) > 1 {
			events <- Event{Type: EventText, Content: fmt.Sprintf("\n**Step %d/%d: %s**\n\n", i+1, len(sorted), step.Name)}
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
			WorkingDir:   req.WorkingDir,
			Message:      msg,
			SystemPrompt: systemPrompt,
		}

		// Collect this step's output so downstream steps can reference it.
		collecting := &collectingEvents{upstream: events}
		if err := a.runStep(ctx, stepReq, harness, systemPrompt, &step, collecting.ch()); err != nil {
			return err
		}
		stepOutputs[step.Name] = collecting.text
	}

	return nil
}

// runStep resolves the harness and runs the agent for a single step.
func (a *ADLAgent) runStep(ctx context.Context, req RunRequest, harness model.ADLHarness, systemPrompt string, step *model.ADLStep, events chan<- Event) error {
	deps := harnessDepsFromADL(a.def, step)
	configDir, err := ProvisionHarnessConfig(a.projectID, harness.Type, deps)
	if err != nil {
		return fmt.Errorf("provision harness config: %w", err)
	}
	req.ConfigDir = configDir
	req.SystemPrompt = systemPrompt
	req.Model = harness.Model

	switch harness.Type {
	case "claude-code", "":
		switch harness.Sandbox {
		case "docker":
			ag, err := a.manager.GetClaudeCodeDocker(a.projectID, harness.Image, req.WorkingDir)
			if err != nil {
				return fmt.Errorf("claude-code docker harness: %w", err)
			}
			return ag.Run(ctx, req, events)
		default:
			if harness.Sandbox == "bubblewrap" {
				bwrap := GetBwrapStatus()
				if !bwrap.Available {
					return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
				}
			}
			ag, err := a.manager.GetAgent(a.projectID, "claude-code", req.WorkingDir, harnessBuiltinConfig(harness))
			if err != nil {
				return fmt.Errorf("claude-code harness: %w", err)
			}
			return ag.Run(ctx, req, events)
		}

	case "pi":
		switch harness.Sandbox {
		case "docker":
			ag, err := a.manager.GetPiDocker(a.projectID, harness.Image, req.WorkingDir)
			if err != nil {
				return fmt.Errorf("pi docker harness: %w", err)
			}
			return ag.Run(ctx, req, events)
		default:
			if harness.Sandbox == "bubblewrap" {
				bwrap := GetBwrapStatus()
				if !bwrap.Available {
					return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
				}
			}
			ag, err := a.manager.GetAgent(a.projectID, "pi", req.WorkingDir, harnessBuiltinConfig(harness))
			if err != nil {
				return fmt.Errorf("pi harness: %w", err)
			}
			return ag.Run(ctx, req, events)
		}

	case "codex":
		switch harness.Sandbox {
		case "docker":
			ag, err := a.manager.GetCodexDocker(a.projectID, harness.Image, req.WorkingDir)
			if err != nil {
				return fmt.Errorf("codex docker harness: %w", err)
			}
			return ag.Run(ctx, req, events)
		default:
			if harness.Sandbox == "bubblewrap" {
				bwrap := GetBwrapStatus()
				if !bwrap.Available {
					return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
				}
			}
			ag, err := a.manager.GetAgent(a.projectID, "codex", req.WorkingDir, harnessBuiltinConfig(harness))
			if err != nil {
				return fmt.Errorf("codex harness: %w", err)
			}
			return ag.Run(ctx, req, events)
		}

	case "opencode":
		switch harness.Sandbox {
		case "docker":
			ag, err := a.manager.GetOpenCodeDocker(a.projectID, harness.Image, req.WorkingDir)
			if err != nil {
				return fmt.Errorf("opencode docker harness: %w", err)
			}
			return ag.Run(ctx, req, events)
		default:
			if harness.Sandbox == "bubblewrap" {
				bwrap := GetBwrapStatus()
				if !bwrap.Available {
					return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
				}
			}
			ag, err := a.manager.GetAgent(a.projectID, "opencode", req.WorkingDir, harnessBuiltinConfig(harness))
			if err != nil {
				return fmt.Errorf("opencode harness: %w", err)
			}
			return ag.Run(ctx, req, events)
		}

	case "docker":
		// External HTTP/SSE agent in a user-managed Docker container.
		ag, err := a.manager.GetAgent(a.projectID, "docker", req.WorkingDir, map[string]any{
			"image":         harness.Image,
			"containerPort": harness.ContainerPort,
		})
		if err != nil {
			return fmt.Errorf("docker harness: %w", err)
		}
		return ag.Run(ctx, req, events)

	case "remote":
		ag, err := a.manager.GetAgent(a.projectID, "remote", req.WorkingDir, map[string]any{
			"host": harness.Host,
			"port": harness.Port,
		})
		if err != nil {
			return fmt.Errorf("remote harness: %w", err)
		}
		return ag.Run(ctx, req, events)

	default:
		return fmt.Errorf("unknown harness type: %q", harness.Type)
	}
}

// buildStepMessage constructs the message sent to a step, injecting upstream outputs.
func buildStepMessage(userMsg string, step model.ADLStep, outputs map[string]string) string {
	if len(step.Inputs) > 0 {
		var b strings.Builder
		for _, inp := range step.Inputs {
			parts := strings.SplitN(inp.From, ".", 2)
			if len(parts) != 2 {
				continue
			}
			stepName := parts[0]
			if out, ok := outputs[stepName]; ok {
				label := inp.As
				if label == "" {
					label = inp.From
				}
				fmt.Fprintf(&b, "## %s\n\n%s\n\n", label, out)
			}
		}
		if b.Len() > 0 {
			return b.String() + userMsg
		}
	} else if len(step.DependsOn) > 0 {
		var b strings.Builder
		for _, dep := range step.DependsOn {
			if out, ok := outputs[dep]; ok {
				fmt.Fprintf(&b, "## Output from %s\n\n%s\n\n", dep, out)
			}
		}
		if b.Len() > 0 {
			return b.String() + userMsg
		}
	}
	return userMsg
}

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
}

func (c *collectingEvents) ch() chan<- Event {
	c.pipe = make(chan Event, 64)
	go func() {
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
