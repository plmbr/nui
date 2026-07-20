// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nui/internal/model"
	"nui/internal/store"

	"gopkg.in/yaml.v3"
)

// SaveDefinitionYAML validates and writes an ADL agent definition to ~/.nui/agents/.
func SaveDefinitionYAML(content string, overwrite bool) (string, error) {
	def, err := ParseDefinitionYAML(content)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(model.ADLAgentID(def))
	if id == "" {
		return "", fmt.Errorf("agent id is required")
	}
	dir, err := store.AgentsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, id+".yaml")
	if _, err := os.Stat(path); err == nil && !overwrite {
		return "", fmt.Errorf("agent %q already exists (set overwrite=true to replace)", id)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	normalized, err := yaml.Marshal(def)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, normalized, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// ParseDefinitionYAML parses and validates agent ADL YAML content.
func ParseDefinitionYAML(content string) (model.ADLDefinition, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return model.ADLDefinition{}, fmt.Errorf("content is required")
	}
	var def model.ADLDefinition
	if err := yaml.Unmarshal([]byte(content), &def); err != nil {
		return model.ADLDefinition{}, fmt.Errorf("parse agent ADL: %w", err)
	}
	model.NormalizeADLDefinition(&def)
	model.NormalizeADLSkills(&def)
	model.NormalizeADLTags(&def)
	if err := model.ValidateADLDefinition(def); err != nil {
		return model.ADLDefinition{}, err
	}
	if err := ValidateOrchestratorRefs(def); err != nil {
		return model.ADLDefinition{}, err
	}
	return def, nil
}
