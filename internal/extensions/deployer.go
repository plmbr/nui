// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"loop/internal/model"
)

// ExtensionAgentDeployer is a named deploy command declared under contributions.aiAssets.agentDeployers.
type ExtensionAgentDeployer struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Command     []string `yaml:"command"`
}

// ResolvedDeployer resolves an extension deployer and its owning extension directory.
type ResolvedDeployer struct {
	Extension *Extension
	Deployer  ExtensionAgentDeployer
	ID        string
}

// DeployRequest is written as one JSON line to a deployer command stdin.
type DeployRequest struct {
	Action     string              `json:"action"`
	DeployerID string              `json:"deployerId"`
	AgentID    string              `json:"agentId"`
	Definition model.ADLDefinition `json:"definition"`
	Assets     DeployAssets        `json:"assets"`
}

// DeployAssets bundles resolved ADL aiAssets for deployer consumption.
type DeployAssets struct {
	Skills     []model.ADLSkill     `json:"skills"`
	MCPServers []model.ADLMCPServer `json:"mcpServers"`
	Rules      []DeployRuleAsset    `json:"rules"`
}

// DeployRuleAsset is a resolved rule body for deploy bundling.
type DeployRuleAsset struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// DeployResponse is read as one JSON line from a deployer command stdout.
type DeployResponse struct {
	OK           bool              `json:"ok"`
	DeploymentID string            `json:"deploymentId,omitempty"`
	Status       string            `json:"status,omitempty"`
	Message      string            `json:"message,omitempty"`
	Error        string            `json:"error,omitempty"`
	Endpoint     *DeployEndpoint   `json:"endpoint,omitempty"`
}

// DeployEndpoint describes a reachable deployed agent service.
type DeployEndpoint struct {
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`
	URL  string `json:"url,omitempty"`
}

// DeployerInfo is the list entry for loop agent deployers and GET /api/agent-deployers.
type DeployerInfo struct {
	ID          string `json:"id"`
	Extension   string `json:"extension"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func validateAgentDeployers(deployers []ExtensionAgentDeployer, extName string) error {
	seen := map[string]bool{}
	for i, d := range deployers {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			return fmt.Errorf("extension %s: agentDeployers[%d]: name is required", extName, i)
		}
		if seen[name] {
			return fmt.Errorf("extension %s: duplicate agentDeployer name %q", extName, name)
		}
		seen[name] = true
		if len(d.Command) == 0 {
			return fmt.Errorf("extension %s: agentDeployers[%q]: command is required", extName, name)
		}
	}
	return nil
}

func expandAgentDeployers(extDir string, deployers []ExtensionAgentDeployer) []ExtensionAgentDeployer {
	if len(deployers) == 0 {
		return nil
	}
	out := make([]ExtensionAgentDeployer, len(deployers))
	copy(out, deployers)
	for i := range out {
		out[i].Command = expandCommand(out[i].Command, extDir)
	}
	return out
}

// ResolveDeployer finds ext:<extension>/<deployer-name>.
func (r *Registry) ResolveDeployer(deployerID string) (ResolvedDeployer, error) {
	extName, deployerName, ok := ParseExtRef(deployerID)
	if !ok {
		return ResolvedDeployer{}, fmt.Errorf("invalid deployer id %q (expected ext:<extension>/<name>)", deployerID)
	}
	r.mu.RLock()
	ext, ok := r.extensions[extName]
	r.mu.RUnlock()
	if !ok {
		return ResolvedDeployer{}, fmt.Errorf("extension %q not found", extName)
	}
	if r.isDisabled(extName) {
		return ResolvedDeployer{}, fmt.Errorf("extension %q is disabled", extName)
	}
	for _, d := range ext.AgentDeployers {
		if d.Name == deployerName {
			return ResolvedDeployer{
				Extension: ext,
				Deployer:  d,
				ID:        DeployerRef(extName, deployerName),
			}, nil
		}
	}
	return ResolvedDeployer{}, fmt.Errorf("agent deployer %q not found in extension %q", deployerName, extName)
}

// AllDeployers returns deployer metadata for installed, enabled extensions.
func (r *Registry) AllDeployers() []DeployerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []DeployerInfo
	for name, ext := range r.extensions {
		if r.isDisabled(name) {
			continue
		}
		for _, d := range ext.AgentDeployers {
			out = append(out, DeployerInfo{
				ID:          DeployerRef(name, d.Name),
				Extension:   name,
				Name:        d.Name,
				Description: d.Description,
			})
		}
	}
	return out
}

// InvokeDeployer runs the deployer command with a JSON request on stdin.
func InvokeDeployer(ref ResolvedDeployer, req DeployRequest) (DeployResponse, error) {
	if len(ref.Deployer.Command) == 0 {
		return DeployResponse{}, fmt.Errorf("deployer %q has no command", ref.ID)
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return DeployResponse{}, err
	}
	cmd := exec.Command(ref.Deployer.Command[0], ref.Deployer.Command[1:]...)
	cmd.Dir = ref.Extension.Dir
	cmd.Env = append(os.Environ(),
		"LOOP_EXTENSION_DIR="+ref.Extension.Dir,
		"LOOP_EXTENSION_NAME="+ref.Extension.Manifest.Name,
	)
	cmd.Stdin = bytes.NewReader(append(payload, '\n'))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return DeployResponse{}, fmt.Errorf("deployer %s: %s", ref.ID, msg)
	}
	line := strings.TrimSpace(stdout.String())
	if line == "" {
		return DeployResponse{}, fmt.Errorf("deployer %s: empty response", ref.ID)
	}
	var resp DeployResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return DeployResponse{}, fmt.Errorf("deployer %s: parse response: %w", ref.ID, err)
	}
	if !resp.OK {
		msg := strings.TrimSpace(resp.Error)
		if msg == "" {
			msg = strings.TrimSpace(resp.Message)
		}
		if msg == "" {
			msg = "deploy failed"
		}
		return resp, fmt.Errorf("%s", msg)
	}
	return resp, nil
}
