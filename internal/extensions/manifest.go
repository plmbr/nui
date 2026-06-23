// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const manifestName = "extension.yaml"

// Manifest is the extension entry point at ~/.loop/extensions/<name>/extension.yaml.
type Manifest struct {
	APIVersion    string         `yaml:"apiVersion"`
	Name          string         `yaml:"name"`
	Version       string         `yaml:"version"`
	DisplayName   string         `yaml:"displayName"`
	Description   string         `yaml:"description"`
	Contributions Contributions  `yaml:"contributions"`
}

// Contributions groups list sources for each contribution type.
type Contributions struct {
	Catalog    *CatalogContribution    `yaml:"catalog,omitempty"`
	Harnesses  *HarnessesContribution  `yaml:"harnesses,omitempty"`
	MCPServers *MCPServersContribution `yaml:"mcpServers,omitempty"`
	Skills     *SkillsContribution     `yaml:"skills,omitempty"`
	Agents     *AgentsContribution     `yaml:"agents,omitempty"`
}

type CatalogContribution struct {
	Command []string `yaml:"command"`
}

type HarnessesContribution struct {
	Source  Source         `yaml:"source"`
	Runtime *RuntimeConfig `yaml:"runtime,omitempty"`
}

type MCPServersContribution struct {
	Source Source `yaml:"source"`
}

type SkillsContribution struct {
	Source Source `yaml:"source"`
}

type AgentsContribution struct {
	Source Source `yaml:"source"`
}

// Source loads contribution lists from a file or command process.
type Source struct {
	File    string   `yaml:"file,omitempty"`
	Command []string `yaml:"command,omitempty"`
}

// RuntimeConfig describes how to run a harness process.
type RuntimeConfig struct {
	Transport string   `yaml:"transport"` // stdio | tcp | http
	Command   []string `yaml:"command"`
	Cwd       string   `yaml:"cwd,omitempty"`
	Port      int      `yaml:"port,omitempty"` // http only
	Host      string   `yaml:"host,omitempty"` // http only
}

// HarnessEntry is one harness in a harnesses list.
type HarnessEntry struct {
	ID          string         `yaml:"id"                    json:"id"`
	DisplayName string         `yaml:"displayName"           json:"displayName"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Runtime     *RuntimeConfig `yaml:"runtime,omitempty"     json:"runtime,omitempty"`
}

func LoadManifest(dir string) (Manifest, error) {
	path := filepath.Join(dir, manifestName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := validateManifest(dir, m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func validateManifest(dir string, m Manifest) error {
	dirName := filepath.Base(dir)
	if m.Name == "" {
		return fmt.Errorf("extension %s: name is required", dirName)
	}
	if m.Name != dirName {
		return fmt.Errorf("extension %s: name %q must match directory name", dirName, m.Name)
	}
	if m.APIVersion != "" && m.APIVersion != "loop.dev/extension/v1" {
		return fmt.Errorf("extension %s: unsupported apiVersion %q", m.Name, m.APIVersion)
	}
	return nil
}

func (s Source) resolved(extDir string, catalogCmd []string) (file string, command []string, err error) {
	if f := strings.TrimSpace(s.File); f != "" {
		return filepath.Join(extDir, f), nil, nil
	}
	if len(s.Command) > 0 {
		return "", expandCommand(s.Command, extDir), nil
	}
	if len(catalogCmd) > 0 {
		return "", expandCommand(catalogCmd, extDir), nil
	}
	return "", nil, fmt.Errorf("no source file or command configured")
}

func expandCommand(cmd []string, extDir string) []string {
	out := make([]string, len(cmd))
	copy(out, cmd)
	for i, part := range out {
		out[i] = strings.ReplaceAll(part, "${LOOP_EXTENSION_DIR}", extDir)
	}
	return out
}

func runtimeCwd(extDir string, rt *RuntimeConfig) string {
	if rt == nil {
		return extDir
	}
	cwd := strings.TrimSpace(rt.Cwd)
	if cwd == "" || cwd == "." {
		return extDir
	}
	if filepath.IsAbs(cwd) {
		return cwd
	}
	return filepath.Join(extDir, cwd)
}
