// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"fmt"
	"os"
	"strings"

	"nui/internal/agent"
	"nui/internal/extensions"
	"nui/internal/model"
	"nui/internal/store"

	"gopkg.in/yaml.v3"
)

// DeployResult is returned after a successful agent deployment.
type DeployResult struct {
	DeploymentID string                     `json:"deploymentId,omitempty"`
	Status       string                     `json:"status,omitempty"`
	Message      string                     `json:"message,omitempty"`
	Endpoint     *extensions.DeployEndpoint `json:"endpoint,omitempty"`
}

// LoadUserAgent loads a user-installed ADL agent by id from ~/.nui/agents/.
func LoadUserAgent(agentID string) (model.ADLDefinition, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return model.ADLDefinition{}, fmt.Errorf("agent id is required")
	}
	dir, err := store.AgentsDir()
	if err != nil {
		return model.ADLDefinition{}, err
	}
	path, err := resolveAgentPath(dir, agentID)
	if err != nil {
		return model.ADLDefinition{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return model.ADLDefinition{}, err
	}
	var def model.ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		return model.ADLDefinition{}, fmt.Errorf("parse agent ADL: %w", err)
	}
	model.NormalizeADLDefinition(&def)
	model.NormalizeADLSkills(&def)
	return def, nil
}

// ListDeployers returns installed agent deployers from the extension registry.
func ListDeployers() ([]extensions.DeployerInfo, error) {
	reg, err := extensions.LoadRegistry()
	if err != nil {
		return nil, err
	}
	out := reg.AllDeployers()
	if out == nil {
		out = []extensions.DeployerInfo{}
	}
	return out, nil
}

// Deploy runs an extension agent deployer for a user-installed agent.
func Deploy(deployerID, agentID string) (DeployResult, error) {
	deployerID = strings.TrimSpace(deployerID)
	agentID = strings.TrimSpace(agentID)
	if deployerID == "" {
		return DeployResult{}, fmt.Errorf("deployer id is required")
	}
	if agentID == "" {
		return DeployResult{}, fmt.Errorf("agent id is required")
	}

	def, err := LoadUserAgent(agentID)
	if err != nil {
		return DeployResult{}, err
	}

	reg, err := extensions.LoadRegistry()
	if err != nil {
		return DeployResult{}, err
	}
	ref, err := reg.ResolveDeployer(deployerID)
	if err != nil {
		return DeployResult{}, err
	}

	assets, err := agent.BuildDeployAssets(def, reg)
	if err != nil {
		return DeployResult{}, err
	}

	resp, err := extensions.InvokeDeployer(ref, extensions.DeployRequest{
		Action:     "deploy",
		DeployerID: ref.ID,
		AgentID:    agentID,
		Definition: def,
		Assets:     assets,
	})
	if err != nil {
		return DeployResult{}, err
	}
	return DeployResult{
		DeploymentID: resp.DeploymentID,
		Status:       resp.Status,
		Message:      resp.Message,
		Endpoint:     resp.Endpoint,
	}, nil
}
