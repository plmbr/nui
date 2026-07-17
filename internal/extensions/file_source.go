// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"nui/internal/model"
)

func loadHarnessesFromFile(path string) ([]HarnessEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Harnesses []HarnessEntry `json:"harnesses" yaml:"harnesses"`
	}
	if err := decodeListFile(data, path, &wrap); err != nil {
		return nil, err
	}
	for i, h := range wrap.Harnesses {
		if strings.TrimSpace(h.ID) == "" {
			return nil, fmt.Errorf("%s: harnesses[%d]: id is required", path, i)
		}
	}
	return wrap.Harnesses, nil
}

func loadMCPServersFromFile(path string) ([]model.ADLMCPServer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		MCPServers []model.ADLMCPServer `json:"mcpServers" yaml:"mcpServers"`
	}
	if err := decodeListFile(data, path, &wrap); err != nil {
		return nil, err
	}
	for i, s := range wrap.MCPServers {
		if strings.TrimSpace(s.Name) == "" && strings.TrimSpace(s.Ref) == "" {
			return nil, fmt.Errorf("%s: mcpServers[%d]: name is required", path, i)
		}
	}
	return wrap.MCPServers, nil
}

func loadSkillsFromFile(path string) ([]model.ADLSkill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Skills []model.ADLSkill `json:"skills" yaml:"skills"`
	}
	if err := decodeListFile(data, path, &wrap); err != nil {
		return nil, err
	}
	for i, s := range wrap.Skills {
		if strings.TrimSpace(s.Name) == "" {
			return nil, fmt.Errorf("%s: skills[%d]: name is required", path, i)
		}
	}
	return wrap.Skills, nil
}

func loadAgentsFromFile(path string) ([]model.ADLDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Agents []model.ADLDefinition `json:"agents" yaml:"agents"`
	}
	if err := decodeListFile(data, path, &wrap); err != nil {
		return nil, err
	}
	for i := range wrap.Agents {
		model.NormalizeADLDefinition(&wrap.Agents[i])
		model.NormalizeADLSkills(&wrap.Agents[i])
		if wrap.Agents[i].ID == "" && wrap.Agents[i].Name == "" {
			return nil, fmt.Errorf("%s: agents[%d]: id or name is required", path, i)
		}
		if err := model.ValidateADLDefinition(wrap.Agents[i]); err != nil {
			return nil, fmt.Errorf("%s: agents[%d]: %w", path, i, err)
		}
	}
	return wrap.Agents, nil
}

func decodeListFile(data []byte, path string, v any) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, v); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, v); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	default:
		if err := json.Unmarshal(data, v); err == nil {
			return nil
		}
		if err := yaml.Unmarshal(data, v); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	}
	return nil
}
