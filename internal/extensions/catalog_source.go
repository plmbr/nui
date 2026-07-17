// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"

	"nui/internal/model"
)

type catalogProvider struct {
	rpc     *StdioRPC
	extDir  string
	extName string
}

func newCatalogProvider(extDir, extName string, command []string) (*catalogProvider, error) {
	fmt.Fprintf(os.Stderr, "[extensions] initializing catalog for %q\n", extName)
	env := append(os.Environ(),
		"NUI_EXTENSION_DIR="+extDir,
		"NUI_EXTENSION_NAME="+extName,
	)
	rpc, err := StartStdioRPC(command, env, extDir)
	if err != nil {
		return nil, err
	}
	p := &catalogProvider{rpc: rpc, extDir: extDir, extName: extName}
	var initResult struct {
		APIVersion string `json:"apiVersion"`
	}
	if err := rpc.Call("extension.initialize", map[string]any{
		"extensionName": extName,
		"extensionDir":  extDir,
	}, &initResult); err != nil {
		rpc.Close()
		return nil, fmt.Errorf("catalog %s initialize: %w", extName, err)
	}
	if initResult.APIVersion != "" {
		fmt.Fprintf(os.Stderr, "[extensions] catalog for %q ready (api %s)\n", extName, initResult.APIVersion)
	} else {
		fmt.Fprintf(os.Stderr, "[extensions] catalog for %q ready\n", extName)
	}
	return p, nil
}

func (p *catalogProvider) ListHarnesses() ([]HarnessEntry, error) {
	var result struct {
		Harnesses []HarnessEntry `json:"harnesses"`
	}
	if err := p.rpc.Call("extension.listHarnesses", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return result.Harnesses, nil
}

func (p *catalogProvider) ListMCPServers() ([]model.ADLMCPServer, error) {
	var result struct {
		MCPServers []model.ADLMCPServer `json:"mcpServers"`
	}
	if err := p.rpc.Call("extension.listMCPServers", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return result.MCPServers, nil
}

func (p *catalogProvider) ListSkills() ([]model.ADLSkill, error) {
	var result struct {
		Skills []model.ADLSkill `json:"skills"`
	}
	if err := p.rpc.Call("extension.listSkills", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return result.Skills, nil
}

func (p *catalogProvider) ListAgents() ([]model.ADLDefinition, error) {
	var result struct {
		Agents []model.ADLDefinition `json:"agents"`
	}
	if err := p.rpc.Call("extension.listAgents", map[string]any{}, &result); err != nil {
		return nil, err
	}
	var valid []model.ADLDefinition
	for _, def := range result.Agents {
		defCopy := def
		model.NormalizeADLDefinition(&defCopy)
		model.NormalizeADLSkills(&defCopy)
		if err := model.ValidateADLDefinition(defCopy); err != nil {
			fmt.Fprintf(os.Stderr, "warn: skip invalid extension agent: %v\n", err)
			continue
		}
		valid = append(valid, defCopy)
	}
	return valid, nil
}

func (p *catalogProvider) Close() error {
	if p.rpc == nil {
		return nil
	}
	return p.rpc.Close()
}
