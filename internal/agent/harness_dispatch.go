// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"

	"loop/internal/extensions"
	"loop/internal/model"
)

type harnessRunner func(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error

var harnessRunners = map[string]harnessRunner{
	"claude-code": runClaudeCodeHarness,
	"pi":          runPiHarness,
	"codex":       runCodexHarness,
	"opencode":    runOpenCodeHarness,
	"docker":      runDockerHarness,
	"remote":      runRemoteHarness,
}

func (a *ADLAgent) dispatchHarness(ctx context.Context, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	harnessType := harness.Type
	if harnessType == "" {
		harnessType = "claude-code"
	}
	if run, ok := harnessRunners[harnessType]; ok {
		return run(ctx, a, req, harness, events)
	}
	if extensions.IsExtRef(harnessType) || (a.manager.registry != nil && a.manager.registry.IsExtensionHarnessAgent(harnessType)) {
		ag, err := a.manager.GetAgent(a.projectID, harnessType, req.WorkingDir, nil)
		if err != nil {
			return fmt.Errorf("extension harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	}
	return fmt.Errorf("unknown harness type: %q", harness.Type)
}

func requireBubblewrap(sandbox string) error {
	if sandbox != "bubblewrap" {
		return nil
	}
	bwrap := GetBwrapStatus()
	if !bwrap.Available {
		return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
	}
	return nil
}

func runClaudeCodeHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	switch harness.Sandbox {
	case "docker":
		ag, err := a.manager.GetClaudeCodeDocker(a.projectID, harness.Image, req.WorkingDir, req.ConfigDir, req.UserScopeHarness)
		if err != nil {
			return fmt.Errorf("claude-code docker harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	default:
		if err := requireBubblewrap(harness.Sandbox); err != nil {
			return err
		}
		ag, err := a.manager.GetAgent(a.projectID, "claude-code", req.WorkingDir, harnessBuiltinConfig(harness))
		if err != nil {
			return fmt.Errorf("claude-code harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	}
}

func runPiHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	switch harness.Sandbox {
	case "docker":
		ag, err := a.manager.GetPiDocker(a.projectID, harness.Image, req.WorkingDir, req.ConfigDir)
		if err != nil {
			return fmt.Errorf("pi docker harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	default:
		if err := requireBubblewrap(harness.Sandbox); err != nil {
			return err
		}
		ag, err := a.manager.GetAgent(a.projectID, "pi", req.WorkingDir, harnessBuiltinConfig(harness))
		if err != nil {
			return fmt.Errorf("pi harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	}
}

func runCodexHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	switch harness.Sandbox {
	case "docker":
		ag, err := a.manager.GetCodexDocker(a.projectID, harness.Image, req.WorkingDir, req.ConfigDir, req.UserScopeHarness)
		if err != nil {
			return fmt.Errorf("codex docker harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	default:
		if err := requireBubblewrap(harness.Sandbox); err != nil {
			return err
		}
		ag, err := a.manager.GetAgent(a.projectID, "codex", req.WorkingDir, harnessBuiltinConfig(harness))
		if err != nil {
			return fmt.Errorf("codex harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	}
}

func runOpenCodeHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	switch harness.Sandbox {
	case "docker":
		ag, err := a.manager.GetOpenCodeDocker(a.projectID, harness.Image, req.WorkingDir, req.ConfigDir)
		if err != nil {
			return fmt.Errorf("opencode docker harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	default:
		if err := requireBubblewrap(harness.Sandbox); err != nil {
			return err
		}
		ag, err := a.manager.GetAgent(a.projectID, "opencode", req.WorkingDir, harnessBuiltinConfig(harness))
		if err != nil {
			return fmt.Errorf("opencode harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	}
}

func runDockerHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	ag, err := a.manager.GetAgent(a.projectID, "docker", req.WorkingDir, map[string]any{
		"image":         harness.Image,
		"containerPort": harness.ContainerPort,
	})
	if err != nil {
		return fmt.Errorf("docker harness: %w", err)
	}
	return ag.Run(ctx, req, events)
}

func runRemoteHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	ag, err := a.manager.GetAgent(a.projectID, "remote", req.WorkingDir, map[string]any{
		"host": harness.Host,
		"port": harness.Port,
	})
	if err != nil {
		return fmt.Errorf("remote harness: %w", err)
	}
	return ag.Run(ctx, req, events)
}
