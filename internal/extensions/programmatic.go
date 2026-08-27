// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"strings"

	"nui/internal/model"
	"nui/internal/store"
)

// ContributionManifest is returned by extension.initialize from a programmatic extension process.
type ContributionManifest struct {
	APIVersion         string                     `json:"apiVersion"`
	Name               string                     `json:"name"`
	Version            string                     `json:"version"`
	DisplayName        string                     `json:"displayName,omitempty"`
	Description        string                     `json:"description,omitempty"`
	Harnesses          []HarnessEntry             `json:"harnesses,omitempty"`
	Agents             []model.ADLDefinition      `json:"agents,omitempty"`
	MCPServers         []model.ADLMCPServer       `json:"mcpServers,omitempty"`
	CustomMCPServers   []ExtensionCustomMCPServer `json:"customMCPServers,omitempty"`
	Skills             []model.ADLSkill           `json:"skills,omitempty"`
	CustomSkills       []ExtensionCustomSkill     `json:"customSkills,omitempty"`
	CustomRules        []ExtensionCustomRule      `json:"rules,omitempty"`
	MentionProviders   []MentionProviderEntry     `json:"mentionProviders,omitempty"`
	HITLChannels       []HITLChannelEntry         `json:"hitlChannels,omitempty"`
	StorageHandlers    []StorageHandlerEntry      `json:"storageHandlers,omitempty"`
	AgentDeployers     []ExtensionAgentDeployer   `json:"agentDeployers,omitempty"`
}

type programmaticHost struct {
	rpc        *ProgrammaticRPC
	extDir     string
	extName    string
	entry      string
	manifest   ContributionManifest
}

func startProgrammaticHost(extDir string, manifest Manifest) (*programmaticHost, error) {
	if manifest.Runtime == nil || len(manifest.Runtime.Command) == 0 {
		return nil, fmt.Errorf("programmatic extension %s: runtime.command is required", manifest.Name)
	}
	entry := resolveInstallEntry(extDir, manifest.Install)
	command := expandRuntimeCommand(manifest.Runtime.Command, extDir, entry)
	fixed := map[string]string{
		"NUI_EXTENSION_DIR":  extDir,
		"NUI_EXTENSION_NAME": manifest.Name,
	}
	if entry != "" {
		fixed["NUI_EXTENSION_ENTRY"] = entry
	}
	env := store.ExtensionProcessEnv(manifest.Name, fixed)
	cwd := runtimeCwd(extDir, manifest.Runtime)
	rpc, err := StartProgrammaticRPC(command, env, cwd)
	if err != nil {
		return nil, fmt.Errorf("start programmatic extension %s: %w", manifest.Name, err)
	}
	host := &programmaticHost{
		rpc:     rpc,
		extDir:  extDir,
		extName: manifest.Name,
		entry:   entry,
	}
	var initResult ContributionManifest
	if err := rpc.Call("extension.initialize", map[string]any{
		"extensionName": manifest.Name,
		"extensionDir":  extDir,
		"apiUrl":        defaultnuiAPIURL(),
	}, &initResult); err != nil {
		_ = rpc.Close()
		return nil, fmt.Errorf("extension %s initialize: %w", manifest.Name, err)
	}
	host.manifest = initResult
	if err := validateProgrammaticStorageHandlers(initResult.StorageHandlers, manifest.Name); err != nil {
		_ = rpc.Close()
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "[extensions] programmatic %q initialized\n", manifest.Name)
	return host, nil
}

func defaultnuiAPIURL() string {
	if v := os.Getenv("NUI_API_URL"); v != "" {
		return v
	}
	return "http://127.0.0.1:8080"
}

func (h *programmaticHost) Close() error {
	if h == nil || h.rpc == nil {
		return nil
	}
	return h.rpc.Close()
}

func applyContributionManifest(ext *Extension, manifest ContributionManifest) {
	if len(manifest.Harnesses) > 0 {
		ext.Harnesses = manifest.Harnesses
	}
	if len(manifest.MCPServers) > 0 {
		ext.MCPServers = manifest.MCPServers
	}
	if len(manifest.Skills) > 0 {
		ext.Skills = manifest.Skills
	}
	if len(manifest.Agents) > 0 {
		for i := range manifest.Agents {
			namespaceAgent(&manifest.Agents[i], ext.Manifest.Name)
		}
		ext.Agents = manifest.Agents
	}
	if len(manifest.CustomMCPServers) > 0 {
		ext.CustomMCPServers = expandCustomMCPServers(ext.Dir, manifest.CustomMCPServers)
	}
	if len(manifest.CustomSkills) > 0 {
		ext.CustomSkills = expandCustomSkills(ext.Dir, manifest.CustomSkills)
	}
	if len(manifest.CustomRules) > 0 {
		ext.CustomRules = expandCustomRules(ext.Dir, manifest.CustomRules)
	}
	if len(manifest.MentionProviders) > 0 {
		ext.MentionProviders = manifest.MentionProviders
	}
	if len(manifest.HITLChannels) > 0 {
		ext.HITLChannels = manifest.HITLChannels
	}
	if len(manifest.StorageHandlers) > 0 {
		ext.StorageHandlers = manifest.StorageHandlers
	}
	if len(manifest.AgentDeployers) > 0 {
		ext.AgentDeployers = expandAgentDeployers(ext.Dir, manifest.AgentDeployers)
	}
	if v := strings.TrimSpace(manifest.Version); v != "" {
		ext.Manifest.Version = v
	}
	if v := strings.TrimSpace(manifest.DisplayName); v != "" {
		ext.Manifest.DisplayName = v
	}
	if v := strings.TrimSpace(manifest.Description); v != "" {
		ext.Manifest.Description = v
	}
}
