// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	containerWorkspace = "/workspaces/loop"
	containerUser      = "loop"
)

// ContainerWorkspace is the path inside the devcontainer where the session workspace is mounted.
func ContainerWorkspace() string {
	return containerWorkspace
}

// SessionConfigMountPath is where Loop mounts provisioned harness config inside the container.
func SessionConfigMountPath() string {
	return fmt.Sprintf("/home/%s/.loop/session-config", containerUser)
}

// DefaultImages maps inner harness types to Loop-managed devcontainer images.
var DefaultImages = map[string]string{
	"claude-code": "loop-devcontainer-claude-code:latest",
	"pi":          "loop-devcontainer-pi:latest",
	"codex":       "loop-devcontainer-codex:latest",
	"opencode":    "loop-devcontainer-opencode:latest",
}

// ProvisionOpts configures Loop-managed devcontainer.json generation.
type ProvisionOpts struct {
	SessionID        string
	InnerHarness     string
	Image            string // optional override
	WorkingDir       string // host path mounted into the container
	SessionConfigDir string // provisioned harness config (MCP, skills, etc.)
}

type devcontainerSpec struct {
	Name            string            `json:"name"`
	Image           string            `json:"image"`
	RemoteUser      string            `json:"remoteUser"`
	WorkspaceFolder string            `json:"workspaceFolder"`
	RemoteEnv       map[string]string `json:"remoteEnv,omitempty"`
	Mounts          []string          `json:"mounts,omitempty"`
	RunArgs         []string          `json:"runArgs,omitempty"`
	OverrideCommand bool              `json:"overrideCommand"`
}

// ManagedWorkspaceFolder returns the host folder passed to devcontainer up.
func ManagedWorkspaceFolder(sessionID string) (string, error) {
	if strings.TrimSpace(sessionID) == "" {
		return "", fmt.Errorf("session id is required")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".loop", "sessions", sessionID), nil
}

// ProvisionSession writes ~/.loop/sessions/<sessionId>/.devcontainer/devcontainer.json.
func ProvisionSession(opts ProvisionOpts) (string, error) {
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		return "", fmt.Errorf("session id is required")
	}
	inner := strings.TrimSpace(opts.InnerHarness)
	if inner == "" {
		return "", fmt.Errorf("inner harness is required")
	}

	managedDir, err := ManagedWorkspaceFolder(sessionID)
	if err != nil {
		return "", err
	}
	devDir := filepath.Join(managedDir, ".devcontainer")
	if err := os.MkdirAll(devDir, 0700); err != nil {
		return "", err
	}

	image := strings.TrimSpace(opts.Image)
	if image == "" {
		image = DefaultImages[inner]
	}
	if image == "" {
		return "", fmt.Errorf("no default devcontainer image for inner harness %q", inner)
	}

	workingDir := strings.TrimSpace(opts.WorkingDir)
	if workingDir == "" {
		return "", fmt.Errorf("working directory is required for devcontainer provisioning")
	}

	mounts := buildMounts(workingDir, opts.SessionConfigDir)
	runArgs := loopbackRunArgs()

	spec := devcontainerSpec{
		Name:            "loop-session-" + sessionID,
		Image:           image,
		RemoteUser:      containerUser,
		WorkspaceFolder: containerWorkspace,
		RemoteEnv: map[string]string{
			"ANTHROPIC_API_KEY":     "${localEnv:ANTHROPIC_API_KEY}",
			"ANTHROPIC_AUTH_TOKEN":  "${localEnv:ANTHROPIC_AUTH_TOKEN}",
			"ANTHROPIC_OAUTH_TOKEN": "${localEnv:ANTHROPIC_OAUTH_TOKEN}",
			"ANTHROPIC_BASE_URL":    "${localEnv:ANTHROPIC_BASE_URL}",
			"OPENAI_API_KEY":        "${localEnv:OPENAI_API_KEY}",
			"OPENAI_BASE_URL":       "${localEnv:OPENAI_BASE_URL}",
		},
		Mounts:          mounts,
		RunArgs:         runArgs,
		OverrideCommand: true,
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", err
	}
	configPath := filepath.Join(devDir, devcontainerJSONName)
	if err := os.WriteFile(configPath, append(data, '\n'), 0644); err != nil {
		return "", err
	}
	return managedDir, nil
}

func buildMounts(workingDir, sessionConfigDir string) []string {
	var mounts []string
	mounts = append(mounts, fmt.Sprintf("source=%s,target=%s,type=bind", workingDir, containerWorkspace))
	sessionConfigDir = strings.TrimSpace(sessionConfigDir)
	if sessionConfigDir != "" {
		mounts = append(mounts, fmt.Sprintf(
			"source=%s,target=/home/%s/.loop/session-config,type=bind",
			sessionConfigDir, containerUser,
		))
	}
	return mounts
}

func loopbackRunArgs() []string {
	var args []string
	for _, envKey := range []string{"ANTHROPIC_BASE_URL", "OPENAI_BASE_URL"} {
		if hosts := loopbackHostArgs(os.Getenv(envKey)); len(hosts) > 0 {
			args = append(args, hosts...)
		}
	}
	return args
}
