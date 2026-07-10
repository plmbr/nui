// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

// Package devcontainer wraps the Dev Container CLI for Loop harness lifecycle.
package devcontainer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultConfigRel     = ".devcontainer/devcontainer.json"
	altConfigRel         = ".devcontainer.json"
	devcontainerJSONName = "devcontainer.json"
)

// UpOpts configures devcontainer up.
type UpOpts struct {
	WorkspaceFolder string
	Config          string // optional relative or absolute path to devcontainer.json
}

// UpResult is parsed from the trailing JSON line of devcontainer up output.
type UpResult struct {
	Outcome               string `json:"outcome"`
	Message               string `json:"message"`
	Description           string `json:"description"`
	ContainerID           string `json:"containerId"`
	RemoteUser            string `json:"remoteUser"`
	RemoteWorkspaceFolder string `json:"remoteWorkspaceFolder"`
}

// Available reports whether the devcontainer CLI is on PATH.
func Available() bool {
	_, err := exec.LookPath("devcontainer")
	return err == nil
}

// DockerAvailable reports whether the Docker daemon is reachable.
func DockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

// HarnessAvailable reports whether Loop can provision and run a devcontainer harness.
func HarnessAvailable() bool {
	return Available() && DockerAvailable()
}

// FindConfig locates a devcontainer.json under workspace.
// Returns the path relative to workspace when possible.
func FindConfig(workspace string) (string, error) {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return "", fmt.Errorf("workspace folder is required")
	}
	candidates := []string{
		filepath.Join(workspace, defaultConfigRel),
		filepath.Join(workspace, altConfigRel),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			if rel, err := filepath.Rel(workspace, p); err == nil {
				return rel, nil
			}
			return p, nil
		}
	}
	return "", fmt.Errorf("no devcontainer.json found in %s", workspace)
}

// ResolveConfigPath returns the devcontainer config path to pass to the CLI.
func ResolveConfigPath(workspace, config string) (string, error) {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return "", fmt.Errorf("workspace folder is required")
	}
	config = strings.TrimSpace(config)
	if config == "" {
		return FindConfig(workspace)
	}
	if filepath.IsAbs(config) {
		if st, err := os.Stat(config); err != nil || st.IsDir() {
			return "", fmt.Errorf("devcontainer config not found: %s", config)
		}
		return config, nil
	}
	abs := filepath.Join(workspace, config)
	if st, err := os.Stat(abs); err != nil || st.IsDir() {
		return "", fmt.Errorf("devcontainer config not found: %s", abs)
	}
	return config, nil
}

// Up runs devcontainer up and returns the parsed result.
// When Config is empty the CLI auto-discovers .devcontainer/devcontainer.json under WorkspaceFolder.
func Up(ctx context.Context, opts UpOpts) (UpResult, error) {
	workspace := strings.TrimSpace(opts.WorkspaceFolder)
	if workspace == "" {
		return UpResult{}, fmt.Errorf("workspace folder is required")
	}

	args := []string{"up", "--workspace-folder", workspace}
	configPath := strings.TrimSpace(opts.Config)
	if configPath != "" {
		args = append(args, "--config", configPath)
	} else if _, err := FindConfig(workspace); err != nil {
		return UpResult{}, err
	}
	cmd := exec.CommandContext(ctx, "devcontainer", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return UpResult{}, formatUpError(err, stdout.String(), stderr.String())
	}

	result, err := parseUpOutput(stdout.String())
	if err != nil {
		return UpResult{}, err
	}
	if result.Outcome != "success" {
		return UpResult{}, upResultError(result)
	}
	if result.ContainerID == "" {
		return UpResult{}, fmt.Errorf("devcontainer up: missing containerId in output")
	}
	return result, nil
}

func formatUpError(runErr error, stdout, stderr string) error {
	if result, err := parseUpOutput(stdout); err == nil {
		if upErr := upResultError(result); upErr != nil {
			return upErr
		}
	}
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = strings.TrimSpace(stdout)
	}
	if msg != "" && !strings.HasPrefix(msg, "@devcontainers/cli") {
		return fmt.Errorf("devcontainer up: %w\n%s", runErr, msg)
	}
	return fmt.Errorf("devcontainer up: %w", runErr)
}

func upResultError(result UpResult) error {
	if result.Outcome == "success" {
		return nil
	}
	if msg := strings.TrimSpace(result.Message); msg != "" {
		return fmt.Errorf("devcontainer up: %s", friendlyUpMessage(msg))
	}
	if desc := strings.TrimSpace(result.Description); desc != "" {
		return fmt.Errorf("devcontainer up: %s", desc)
	}
	if result.Outcome != "" {
		return fmt.Errorf("devcontainer up failed: outcome=%q", result.Outcome)
	}
	return fmt.Errorf("devcontainer up failed")
}

func friendlyUpMessage(msg string) string {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "failed to connect") || strings.Contains(lower, "docker.sock") || strings.Contains(msg, "docker ps") {
		return "Docker is not running or not reachable. Start Docker Desktop and retry."
	}
	if strings.Contains(lower, "no such image") || strings.Contains(lower, "pull access denied") || strings.Contains(lower, "manifest unknown") {
		return msg + "\nLoop auto-builds default devcontainer images on first use; check Docker build output above for errors."
	}
	return msg
}

func parseUpOutput(stdout string) (UpResult, error) {
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || line[0] != '{' {
			continue
		}
		var result UpResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		if result.Outcome != "" || result.ContainerID != "" {
			return result, nil
		}
	}
	return UpResult{}, fmt.Errorf("devcontainer up: no JSON result in output")
}

// ResolveHostPort maps a container port to a host port via docker port.
func ResolveHostPort(containerID string, containerPort int) (int, error) {
	if containerPort <= 0 {
		return 0, fmt.Errorf("containerPort must be positive")
	}
	out, err := exec.Command("docker", "port", containerID, strconv.Itoa(containerPort)).Output()
	if err != nil {
		return 0, fmt.Errorf("docker port: %w", err)
	}
	return parseDockerPort(strings.TrimSpace(string(out)))
}

func parseDockerPort(s string) (int, error) {
	lines := strings.Split(s, "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" && len(lines) > 1 {
		last = strings.TrimSpace(lines[len(lines)-2])
	}
	idx := strings.LastIndex(last, ":")
	if idx < 0 {
		return 0, fmt.Errorf("unexpected docker port output: %q", s)
	}
	port, err := strconv.Atoi(last[idx+1:])
	if err != nil {
		return 0, fmt.Errorf("parse docker port %q: %w", s, err)
	}
	return port, nil
}

// ContainerRunning reports whether a Docker container is still running.
func ContainerRunning(containerID string) bool {
	if strings.TrimSpace(containerID) == "" {
		return false
	}
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerID).Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// Stop stops a dev container by container ID.
func Stop(containerID string) error {
	if strings.TrimSpace(containerID) == "" {
		return nil
	}
	return exec.Command("docker", "stop", containerID).Run()
}
