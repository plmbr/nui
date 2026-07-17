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

// Manifest is the extension entry point at ~/.nui/extensions/<name>/extension.yaml.
type Manifest struct {
	APIVersion    string         `yaml:"apiVersion"`
	Name          string         `yaml:"name"`
	Version       string         `yaml:"version"`
	DisplayName   string         `yaml:"displayName"`
	Description   string         `yaml:"description"`
	Kind          string         `yaml:"kind,omitempty"` // declarative | programmatic
	Runtime       *RuntimeConfig `yaml:"runtime,omitempty"`
	Install       *InstallConfig `yaml:"install,omitempty"`
	MCPServers    []ExtensionCustomMCPServer `yaml:"mcpServers,omitempty"` // deprecated: use contributions.aiAssets.mcpServers
	Contributions Contributions    `yaml:"contributions"`
}

// InstallConfig records how a programmatic extension package was installed (CLI provenance).
type InstallConfig struct {
	Source string `yaml:"source"`
	Type   string `yaml:"type"` // npm | pip | go | git | dir | zip
	Entry  string `yaml:"entry"`
	Root   string `yaml:"root,omitempty"`
}

func (m Manifest) IsProgrammatic() bool {
	return strings.EqualFold(strings.TrimSpace(m.Kind), "programmatic")
}

// Contributions groups list sources for each contribution type.
type Contributions struct {
	AIAssets   *AIAssetsContribution   `yaml:"aiAssets,omitempty"`
	Catalog    *CatalogContribution    `yaml:"catalog,omitempty"`
	Harnesses  *HarnessesContribution  `yaml:"harnesses,omitempty"`
	MCPServers *MCPServersContribution `yaml:"mcpServers,omitempty"` // deprecated: use catalog.mcpServers
	Skills     *SkillsContribution     `yaml:"skills,omitempty"`     // deprecated: use catalog.skills
	Agents     *AgentsContribution     `yaml:"agents,omitempty"`
	Mentions      *MentionProvidersContribution `yaml:"mentionProviders,omitempty"`
	HITLChannels  *HITLChannelsContribution     `yaml:"hitlChannels,omitempty"`
	Storage       *StorageContribution          `yaml:"storage,omitempty"`
}

// AIAssetsContribution declares custom MCP servers, skills, rules, and agent deployers.
type AIAssetsContribution struct {
	MCPServers     []ExtensionCustomMCPServer  `yaml:"mcpServers,omitempty"`
	Skills         []ExtensionCustomSkill      `yaml:"skills,omitempty"`
	Rules          []ExtensionCustomRule       `yaml:"rules,omitempty"`
	AgentDeployers []ExtensionAgentDeployer    `yaml:"agentDeployers,omitempty"`
}

type CatalogContribution struct {
	Command    []string                `yaml:"command,omitempty"`
	MCPServers *MCPServersContribution `yaml:"mcpServers,omitempty"`
	Skills     *SkillsContribution     `yaml:"skills,omitempty"`
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
	return loadManifestFromDir(dir, true)
}

func loadManifestForInstall(dir string) (Manifest, error) {
	return loadManifestFromDir(dir, false)
}

func loadManifestFromDir(dir string, matchDirName bool) (Manifest, error) {
	path := filepath.Join(dir, manifestName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := validateManifest(dir, m, matchDirName); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func validateManifest(dir string, m Manifest, matchDirName bool) error {
	dirName := filepath.Base(dir)
	if m.Name == "" {
		return fmt.Errorf("extension %s: name is required", dirName)
	}
	if matchDirName && m.Name != dirName {
		return fmt.Errorf("extension %s: name %q must match directory name", dirName, m.Name)
	}
	if m.APIVersion != "" && m.APIVersion != "nui.plmbr.dev/extension/v1" {
		return fmt.Errorf("extension %s: unsupported apiVersion %q", m.Name, m.APIVersion)
	}
	if m.IsProgrammatic() {
		if m.Runtime == nil || len(m.Runtime.Command) == 0 {
			return fmt.Errorf("extension %s: programmatic extensions require runtime.command", m.Name)
		}
		if transport := strings.TrimSpace(m.Runtime.Transport); transport != "" && transport != "stdio" {
			return fmt.Errorf("extension %s: programmatic extensions only support runtime.transport stdio", m.Name)
		}
		return nil
	}
	if err := validateCustomMCPServers(m.aiAssetsMCPServers(), m.Name); err != nil {
		return err
	}
	if err := validateCustomSkills(m.aiAssetsSkills(), m.Name); err != nil {
		return err
	}
	if err := validateCustomRules(m.aiAssetsRules(), m.Name); err != nil {
		return err
	}
	if err := validateAgentDeployers(m.aiAssetsAgentDeployers(), m.Name); err != nil {
		return err
	}
	return nil
}

func (m Manifest) aiAssetsMCPServers() []ExtensionCustomMCPServer {
	if m.Contributions.AIAssets != nil && len(m.Contributions.AIAssets.MCPServers) > 0 {
		return m.Contributions.AIAssets.MCPServers
	}
	return m.MCPServers
}

func (m Manifest) aiAssetsSkills() []ExtensionCustomSkill {
	if m.Contributions.AIAssets != nil {
		return m.Contributions.AIAssets.Skills
	}
	return nil
}

func (m Manifest) aiAssetsRules() []ExtensionCustomRule {
	if m.Contributions.AIAssets != nil {
		return m.Contributions.AIAssets.Rules
	}
	return nil
}

func (m Manifest) aiAssetsAgentDeployers() []ExtensionAgentDeployer {
	if m.Contributions.AIAssets != nil {
		return m.Contributions.AIAssets.AgentDeployers
	}
	return nil
}

// catalogMCPSource returns the catalog MCP list source, falling back to legacy top-level key.
func (m Manifest) catalogMCPSource() *MCPServersContribution {
	if m.Contributions.Catalog != nil && m.Contributions.Catalog.MCPServers != nil {
		return m.Contributions.Catalog.MCPServers
	}
	return m.Contributions.MCPServers
}

// catalogSkillsSource returns the catalog skills list source, falling back to legacy top-level key.
func (m Manifest) catalogSkillsSource() *SkillsContribution {
	if m.Contributions.Catalog != nil && m.Contributions.Catalog.Skills != nil {
		return m.Contributions.Catalog.Skills
	}
	return m.Contributions.Skills
}

// catalogCommand returns the shared catalog RPC command when configured.
func (m Manifest) catalogCommand() []string {
	if m.Contributions.Catalog != nil {
		return m.Contributions.Catalog.Command
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
	return expandRuntimeCommand(cmd, extDir, "")
}

func expandRuntimeCommand(cmd []string, extDir, entry string) []string {
	out := make([]string, len(cmd))
	copy(out, cmd)
	for i, part := range out {
		part = strings.ReplaceAll(part, "${NUI_EXTENSION_DIR}", extDir)
		if entry != "" {
			part = strings.ReplaceAll(part, "${NUI_EXTENSION_ENTRY}", entry)
		}
		out[i] = part
	}
	return out
}

func resolveInstallEntry(extDir string, install *InstallConfig) string {
	if install == nil {
		return ""
	}
	entry := strings.TrimSpace(install.Entry)
	if entry == "" {
		return ""
	}
	entry = strings.ReplaceAll(entry, "${NUI_EXTENSION_DIR}", extDir)
	if filepath.IsAbs(entry) {
		return entry
	}
	return filepath.Clean(filepath.Join(extDir, entry))
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
