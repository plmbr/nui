// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"fmt"
	"strings"

	"nui/internal/devcontainer"
	"nui/internal/extensions"
	"nui/internal/model"
)

type harnessRunner func(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error

var harnessRunners = map[string]harnessRunner{
	"claude-code":  runClaudeCodeHarness,
	"pi":           runPiHarness,
	"codex":        runCodexHarness,
	"opencode":     runOpenCodeHarness,
	"docker":       runDockerHarness,
	"devcontainer": runDevcontainerHarness,
	"remote":       runRemoteHarness,
	"api":          runAPIHarness,
}

func (a *ADLAgent) harnessProjectID(req RunRequest, harness model.ADLHarness) string {
	if !req.Ephemeral {
		return a.projectID
	}
	// Docker and devcontainer share one container per nui session; ephemeral turns use flags instead.
	if harness.Sandbox == "docker" || harness.Type == "devcontainer" {
		return a.projectID
	}
	return EphemeralProjectID(a.projectID)
}

func (a *ADLAgent) dispatchHarness(ctx context.Context, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	if req.Ephemeral {
		req.SessionID = ""
	}
	if a.manager != nil && a.manager.testHarnessRun != nil {
		return a.manager.testHarnessRun(ctx, req, events)
	}
	harnessType := harness.Type
	if harnessType == "" {
		harnessType = "claude-code"
	}
	if run, ok := harnessRunners[harnessType]; ok {
		return run(ctx, a, req, harness, events)
	}
	if extensions.IsExtRef(harnessType) || (a.manager.registry != nil && a.manager.registry.IsExtensionHarnessAgent(harnessType)) {
		ag, err := a.manager.GetAgent(a.harnessProjectID(req, harness), harnessType, req.WorkingDir, nil)
		if err != nil {
			if extensions.IsExtRef(harnessType) && a.manager.registry != nil {
				if _, ok := a.manager.registry.FindAgent(harnessType); ok {
					return fmt.Errorf("harness.type %q is an extension agent id, not a harness type; use harness.type: api or select that agent directly", harnessType)
				}
			}
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
	projectID := a.harnessProjectID(req, harness)
	switch harness.Sandbox {
	case "docker":
		ag, err := a.manager.GetClaudeCodeDocker(projectID, harness.Image, req.WorkingDir, req.ConfigDir, req.UserScopeHarness)
		if err != nil {
			return fmt.Errorf("claude-code docker harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	default:
		if err := requireBubblewrap(harness.Sandbox); err != nil {
			return err
		}
		ag, err := a.manager.GetAgent(projectID, "claude-code", req.WorkingDir, harnessBuiltinConfig(harness))
		if err != nil {
			return fmt.Errorf("claude-code harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	}
}

func runPiHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	projectID := a.harnessProjectID(req, harness)
	switch harness.Sandbox {
	case "docker":
		ag, err := a.manager.GetPiDocker(projectID, harness.Image, req.WorkingDir, req.ConfigDir)
		if err != nil {
			return fmt.Errorf("pi docker harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	default:
		if err := requireBubblewrap(harness.Sandbox); err != nil {
			return err
		}
		ag, err := a.manager.GetAgent(projectID, "pi", req.WorkingDir, harnessBuiltinConfig(harness))
		if err != nil {
			return fmt.Errorf("pi harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	}
}

func runCodexHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	projectID := a.harnessProjectID(req, harness)
	switch harness.Sandbox {
	case "docker":
		ag, err := a.manager.GetCodexDocker(projectID, harness.Image, req.WorkingDir, req.ConfigDir, req.UserScopeHarness)
		if err != nil {
			return fmt.Errorf("codex docker harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	default:
		if err := requireBubblewrap(harness.Sandbox); err != nil {
			return err
		}
		ag, err := a.manager.GetAgent(projectID, "codex", req.WorkingDir, harnessBuiltinConfig(harness))
		if err != nil {
			return fmt.Errorf("codex harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	}
}

func runOpenCodeHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	projectID := a.harnessProjectID(req, harness)
	switch harness.Sandbox {
	case "docker":
		ag, err := a.manager.GetOpenCodeDocker(projectID, harness.Image, req.WorkingDir, req.ConfigDir)
		if err != nil {
			return fmt.Errorf("opencode docker harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	default:
		if err := requireBubblewrap(harness.Sandbox); err != nil {
			return err
		}
		ag, err := a.manager.GetAgent(projectID, "opencode", req.WorkingDir, harnessBuiltinConfig(harness))
		if err != nil {
			return fmt.Errorf("opencode harness: %w", err)
		}
		return ag.Run(ctx, req, events)
	}
}

func runDockerHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	projectID := a.harnessProjectID(req, harness)
	ag, err := a.manager.GetAgent(projectID, "docker", req.WorkingDir, map[string]any{
		"image":         harness.Image,
		"containerPort": harness.ContainerPort,
	})
	if err != nil {
		return fmt.Errorf("docker harness: %w", err)
	}
	return ag.Run(ctx, req, events)
}

func runDevcontainerHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	projectID := a.harnessProjectID(req, harness)
	inner := strings.TrimSpace(harness.InnerHarness)
	if inner == "" {
		return fmt.Errorf("devcontainer harness requires innerHarness")
	}
	ag, err := a.manager.GetDevcontainerAgent(projectID, inner, req.WorkingDir, req.ConfigDir, harness.Image)
	if err != nil {
		return fmt.Errorf("devcontainer harness: %w", err)
	}
	req.WorkingDir = devcontainer.ContainerWorkspace()
	req.ConfigDir = devcontainer.SessionConfigMountPath()
	return ag.Run(ctx, req, events)
}

func runRemoteHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	projectID := a.harnessProjectID(req, harness)
	ag, err := a.manager.GetAgent(projectID, "remote", req.WorkingDir, map[string]any{
		"host": harness.Host,
		"port": harness.Port,
	})
	if err != nil {
		return fmt.Errorf("remote harness: %w", err)
	}
	return ag.Run(ctx, req, events)
}

func runAPIHarness(ctx context.Context, a *ADLAgent, req RunRequest, harness model.ADLHarness, events chan<- Event) error {
	ag := &APIHarnessAgent{Harness: harness, Manager: a.manager}
	return ag.Run(ctx, req, events)
}
