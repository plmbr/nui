// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"loop/internal/model"
	"loop/internal/store"
)

// Extension holds a loaded extension and its resolved contribution lists.
type Extension struct {
	Dir      string
	Manifest Manifest

	Harnesses        []HarnessEntry
	MCPServers       []model.ADLMCPServer
	Skills           []model.ADLSkill
	Agents           []model.ADLDefinition
	CustomMCPServers   []ExtensionCustomMCPServer
	CustomSkills       []ExtensionCustomSkill
	CustomRules        []ExtensionCustomRule
	AgentDeployers     []ExtensionAgentDeployer
	MentionProviders   []MentionProviderEntry
	HITLChannels       []HITLChannelEntry
	StorageHandlers    []StorageHandlerEntry
	mentionRuntime   *RuntimeConfig
	hitlRuntime      *RuntimeConfig
	storageRuntime   *RuntimeConfig

	defaultRuntime   *RuntimeConfig
	programmaticHost *programmaticHost
	resolvedEntry    string
}

// HarnessRef resolves a harness entry and its runtime for an extension harness agent id.
type HarnessRef struct {
	Extension *Extension
	Entry     HarnessEntry
	Runtime   RuntimeConfig
	AgentID   string
}

// Registry indexes all installed extensions.
type Registry struct {
	mu                sync.RWMutex
	extensions        map[string]*Extension // name → extension
	catalogs          []*catalogProvider
	programmaticHosts []*programmaticHost
	mentionCache      *mentionClientCache
	storageCache      *storageClientCache
	loadErrors        []string
}

// Default is the process-wide extension registry, set at server startup.
var Default *Registry

// LoadRegistry scans ~/.loop/extensions/*/extension.yaml and resolves contribution lists.
func LoadRegistry() (*Registry, error) {
	dir, err := store.ExtensionsDir()
	if err != nil {
		return nil, err
	}
	reg, err := scanExtensions(dir)
	if err != nil {
		return nil, err
	}
	Default = reg
	return reg, nil
}

func scanExtensions(dir string) (*Registry, error) {
	fmt.Fprintf(os.Stderr, "[extensions] scanning %s\n", dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	reg := &Registry{extensions: map[string]*Extension{}, mentionCache: newMentionClientCache(), storageCache: newStorageClientCache()}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		extDir := filepath.Join(dir, e.Name())
		fmt.Fprintf(os.Stderr, "[extensions] loading %q\n", e.Name())
		ext, err := loadExtension(extDir, reg)
		if err != nil {
			reg.loadErrors = append(reg.loadErrors, fmt.Sprintf("%s: %v", e.Name(), err))
			fmt.Fprintf(os.Stderr, "[extensions] skip %q: %v\n", e.Name(), err)
			continue
		}
		reg.extensions[ext.Manifest.Name] = ext
		logExtensionLoaded(ext)
	}
	logRegistrySummary(reg)
	return reg, nil
}

func logExtensionLoaded(ext *Extension) {
	var parts []string
	addCount := func(n int, label string) {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, label))
		}
	}
	addCount(len(ext.Harnesses), "harnesses")
	addCount(len(ext.MCPServers)+len(ext.CustomMCPServers), "mcp servers")
	addCount(len(ext.Skills)+len(ext.CustomSkills), "skills")
	addCount(len(ext.CustomRules), "rules")
	addCount(len(ext.AgentDeployers), "agent deployers")
	addCount(len(ext.Agents), "agents")
	addCount(len(ext.MentionProviders), "mention providers")
	addCount(len(ext.StorageHandlers), "storage handlers")

	summary := "no contributions"
	if len(parts) > 0 {
		summary = strings.Join(parts, ", ")
	}
	if v := ext.Manifest.Version; v != "" {
		fmt.Fprintf(os.Stderr, "[extensions] loaded %q v%s: %s\n", ext.Manifest.Name, v, summary)
		return
	}
	fmt.Fprintf(os.Stderr, "[extensions] loaded %q: %s\n", ext.Manifest.Name, summary)
}

func logRegistrySummary(reg *Registry) {
	n := len(reg.extensions)
	skipped := len(reg.loadErrors)
	switch {
	case n == 0 && skipped == 0:
		fmt.Fprintln(os.Stderr, "[extensions] ready: no extensions installed")
	case skipped > 0:
		fmt.Fprintf(os.Stderr, "[extensions] ready: %d loaded, %d skipped\n", n, skipped)
	default:
		fmt.Fprintf(os.Stderr, "[extensions] ready: %d extension(s) loaded\n", n)
	}
}

func loadExtension(extDir string, reg *Registry) (*Extension, error) {
	manifest, err := LoadManifest(extDir)
	if err != nil {
		return nil, err
	}
	ext := &Extension{
		Dir:           extDir,
		Manifest:      manifest,
		resolvedEntry: resolveInstallEntry(extDir, manifest.Install),
	}
	if manifest.IsProgrammatic() {
		return loadProgrammaticExtension(ext, reg)
	}
	return loadDeclarativeExtension(ext, reg)
}

func loadProgrammaticExtension(ext *Extension, reg *Registry) (*Extension, error) {
	rt := *ext.Manifest.Runtime
	if rt.Transport == "" {
		rt.Transport = "stdio"
	}
	ext.defaultRuntime = &rt
	host, err := startProgrammaticHost(ext.Dir, ext.Manifest)
	if err != nil {
		return nil, err
	}
	ext.programmaticHost = host
	reg.programmaticHosts = append(reg.programmaticHosts, host)
	applyContributionManifest(ext, host.manifest)
	return ext, nil
}

func loadDeclarativeExtension(ext *Extension, reg *Registry) (*Extension, error) {
	manifest := ext.Manifest
	if len(manifest.MCPServers) > 0 {
		fmt.Fprintf(os.Stderr, "[extensions] %s: root mcpServers is deprecated; use contributions.aiAssets.mcpServers\n", manifest.Name)
	}
	extDir := ext.Dir
	ext.CustomMCPServers = expandCustomMCPServers(extDir, manifest.aiAssetsMCPServers())
	ext.CustomSkills = expandCustomSkills(extDir, manifest.aiAssetsSkills())
	ext.CustomRules = expandCustomRules(extDir, manifest.aiAssetsRules())
	ext.AgentDeployers = expandAgentDeployers(extDir, manifest.aiAssetsAgentDeployers())
	if manifest.Contributions.MCPServers != nil {
		fmt.Fprintf(os.Stderr, "[extensions] %s: contributions.mcpServers is deprecated; use contributions.catalog.mcpServers\n", manifest.Name)
	}
	if manifest.Contributions.Skills != nil {
		fmt.Fprintf(os.Stderr, "[extensions] %s: contributions.skills is deprecated; use contributions.catalog.skills\n", manifest.Name)
	}
	if manifest.Contributions.Harnesses != nil && manifest.Contributions.Harnesses.Runtime != nil {
		rt := *manifest.Contributions.Harnesses.Runtime
		ext.defaultRuntime = &rt
	}

	var catalogCmd []string
	catalogCmd = manifest.catalogCommand()
	var catalog *catalogProvider

	resolveCatalog := func() (*catalogProvider, error) {
		if catalog != nil {
			return catalog, nil
		}
		if len(catalogCmd) == 0 {
			return nil, fmt.Errorf("catalog command not configured")
		}
		p, err := newCatalogProvider(extDir, manifest.Name, expandCommand(catalogCmd, extDir))
		if err != nil {
			return nil, err
		}
		catalog = p
		reg.catalogs = append(reg.catalogs, p)
		return p, nil
	}

	if c := manifest.Contributions.Harnesses; c != nil {
		list, err := loadContributionList(extDir, manifest.Name, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]HarnessEntry, error) { return loadHarnessesFromFile(file) },
			func(p *catalogProvider) ([]HarnessEntry, error) { return p.ListHarnesses() },
		)
		if err != nil {
			return nil, fmt.Errorf("harnesses: %w", err)
		}
		ext.Harnesses = list
	}

	if c := manifest.catalogMCPSource(); c != nil {
		list, err := loadContributionList(extDir, manifest.Name, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]model.ADLMCPServer, error) { return loadMCPServersFromFile(file) },
			func(p *catalogProvider) ([]model.ADLMCPServer, error) { return p.ListMCPServers() },
		)
		if err != nil {
			return nil, fmt.Errorf("catalog mcpServers: %w", err)
		}
		ext.MCPServers = list
	}

	if c := manifest.catalogSkillsSource(); c != nil {
		list, err := loadContributionList(extDir, manifest.Name, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]model.ADLSkill, error) { return loadSkillsFromFile(file) },
			func(p *catalogProvider) ([]model.ADLSkill, error) { return p.ListSkills() },
		)
		if err != nil {
			return nil, fmt.Errorf("catalog skills: %w", err)
		}
		ext.Skills = list
	}

	if c := manifest.Contributions.Agents; c != nil {
		list, err := loadContributionList(extDir, manifest.Name, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]model.ADLDefinition, error) { return loadAgentsFromFile(file) },
			func(p *catalogProvider) ([]model.ADLDefinition, error) { return p.ListAgents() },
		)
		if err != nil {
			return nil, fmt.Errorf("agents: %w", err)
		}
		for i := range list {
			namespaceAgent(&list[i], manifest.Name)
		}
		ext.Agents = list
	}

	if c := manifest.Contributions.Mentions; c != nil {
		list, err := loadContributionList(extDir, manifest.Name, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]MentionProviderEntry, error) { return loadMentionProvidersFromFile(file) },
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("mentionProviders: %w", err)
		}
		ext.MentionProviders = list
		if c.Runtime != nil {
			rt := *c.Runtime
			ext.mentionRuntime = &rt
		}
	}

	if c := manifest.Contributions.HITLChannels; c != nil {
		list, err := loadContributionList(extDir, manifest.Name, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]HITLChannelEntry, error) { return loadHITLChannelsFromFile(file) },
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("hitlChannels: %w", err)
		}
		ext.HITLChannels = list
		if c.Runtime != nil {
			rt := *c.Runtime
			ext.hitlRuntime = &rt
		}
	}

	if c := manifest.Contributions.Storage; c != nil {
		list, err := loadContributionList(extDir, manifest.Name, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]StorageHandlerEntry, error) { return loadStorageHandlersFromFile(file) },
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("storage: %w", err)
		}
		ext.StorageHandlers = list
		if c.Runtime != nil {
			rt := *c.Runtime
			ext.storageRuntime = &rt
		}
	}

	return ext, nil
}

func loadContributionList[T any](
	extDir string,
	extName string,
	source Source,
	catalogCmd []string,
	resolveCatalog func() (*catalogProvider, error),
	fromFile func(string) ([]T, error),
	fromCatalog func(*catalogProvider) ([]T, error),
) ([]T, error) {
	file, listCmd, err := source.resolved(extDir, catalogCmd)
	if err != nil {
		return nil, err
	}
	if file != "" {
		return fromFile(file)
	}
	if len(listCmd) > 0 {
		if fromCatalog == nil {
			return nil, fmt.Errorf("extension %s: dynamic list source.command is not supported for this contribution type", extName)
		}
		p, err := newCatalogProvider(extDir, extName, listCmd)
		if err != nil {
			return nil, err
		}
		defer p.Close()
		return fromCatalog(p)
	}
	p, err := resolveCatalog()
	if err != nil {
		return nil, err
	}
	return fromCatalog(p)
}

func namespaceAgent(def *model.ADLDefinition, extName string) {
	localID := def.ID
	if localID == "" {
		localID = model.ADLAgentID(*def)
	}
	def.ID = AgentAgentID(extName, localID)
}

// Reload rescans the extensions directory.
func (r *Registry) Reload() error {
	fmt.Fprintln(os.Stderr, "[extensions] reloading")
	r.Shutdown()
	dir, err := store.ExtensionsDir()
	if err != nil {
		return err
	}
	next, err := scanExtensions(dir)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.extensions = next.extensions
	r.catalogs = next.catalogs
	r.loadErrors = next.loadErrors
	r.mentionCache = next.mentionCache
	r.storageCache = next.storageCache
	r.mu.Unlock()
	Default = r
	return nil
}

// Shutdown stops catalog provider processes and programmatic extension hosts.
func (r *Registry) Shutdown() {
	r.mu.Lock()
	catalogs := r.catalogs
	hosts := r.programmaticHosts
	cache := r.mentionCache
	storageCache := r.storageCache
	r.mu.Unlock()
	for _, c := range catalogs {
		_ = c.Close()
	}
	for _, h := range hosts {
		_ = h.Close()
	}
	if cache != nil {
		cache.closeAll()
	}
	if storageCache != nil {
		storageCache.closeAll()
	}
}

func (r *Registry) LoadErrors() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.loadErrors))
	copy(out, r.loadErrors)
	return out
}

// All returns all loaded extensions sorted by name.
func (r *Registry) All() []*Extension {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Extension, 0, len(r.extensions))
	for _, e := range r.extensions {
		out = append(out, e)
	}
	return out
}

// Get returns an extension by name.
func (r *Registry) Get(name string) (*Extension, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.extensions[name]
	return e, ok
}

func disabledExtensions() map[string]bool {
	settings, _ := store.LoadSettings()
	out := map[string]bool{}
	for _, name := range settings.DisabledExtensions {
		out[name] = true
	}
	return out
}

func (r *Registry) isDisabled(name string) bool {
	return disabledExtensions()[name]
}

// AllAgents returns extension-contributed ADL agents.
func (r *Registry) AllAgents() []model.ADLDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []model.ADLDefinition
	for name, ext := range r.extensions {
		if r.isDisabled(name) {
			continue
		}
		out = append(out, ext.Agents...)
	}
	return out
}

// ResolveHarness finds a harness by global agent id ext:<ext>/<harness-id>.
func (r *Registry) ResolveHarness(agentID string) (HarnessRef, bool) {
	extName, harnessID, ok := ParseExtRef(agentID)
	if !ok {
		return HarnessRef{}, false
	}
	r.mu.RLock()
	ext, ok := r.extensions[extName]
	r.mu.RUnlock()
	if !ok || r.isDisabled(extName) {
		return HarnessRef{}, false
	}
	for _, h := range ext.Harnesses {
		if h.ID != harnessID {
			continue
		}
		rt, ok := ext.resolveHarnessRuntime(h)
		if !ok {
			return HarnessRef{}, false
		}
		return HarnessRef{
			Extension: ext,
			Entry:     h,
			Runtime:   rt,
			AgentID:   agentID,
		}, true
	}
	return HarnessRef{}, false
}

func (ext *Extension) resolveHarnessRuntime(h HarnessEntry) (RuntimeConfig, bool) {
	if ext.programmaticHost != nil && ext.defaultRuntime != nil {
		return *ext.defaultRuntime, true
	}
	if h.Runtime != nil {
		return *h.Runtime, true
	}
	if ext.defaultRuntime != nil {
		return *ext.defaultRuntime, true
	}
	return RuntimeConfig{}, false
}

// ProgrammaticHost returns the shared host process for programmatic extensions.
func (ext *Extension) NewProgrammaticHarnessAgent(agentName, harnessID, projectID string) *ProgrammaticHarnessAgent {
	if ext.programmaticHost == nil {
		return nil
	}
	return NewProgrammaticHarnessAgent(ext.programmaticHost, agentName, harnessID, projectID)
}

// IsProgrammatic reports whether the extension runs as a programmatic package.
func (ext *Extension) IsProgrammatic() bool {
	return ext.programmaticHost != nil
}

// IsExtensionHarnessAgent reports whether agentType is ext:<ext>/<harness-id> for a known harness.
func (r *Registry) IsExtensionHarnessAgent(agentType string) bool {
	_, ok := r.ResolveHarness(agentType)
	return ok
}

// HarnessOnlyAgentTypes returns synthetic agent types for harnesses without a matching agent entry.
func (r *Registry) HarnessOnlyAgentTypes() []model.ADLDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agentHarness := map[string]bool{}
	for _, ext := range r.extensions {
		for _, a := range ext.Agents {
			agentHarness[a.Harness.Type] = true
		}
	}
	var out []model.ADLDefinition
	for name, ext := range r.extensions {
		if r.isDisabled(name) {
			continue
		}
		for _, h := range ext.Harnesses {
			agentID := HarnessAgentID(ext.Manifest.Name, h.ID)
			if agentHarness[agentID] {
				continue
			}
			label := h.DisplayName
			if label == "" {
				label = h.ID
			}
			out = append(out, model.ADLDefinition{
				ID:          agentID,
				Name:        label,
				Description: h.Description,
				Harness:     model.ADLHarness{Type: agentID},
			})
		}
	}
	return out
}

// ExtensionInfo is the API shape for GET /api/extensions.
type ExtensionInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	Description string   `json:"description,omitempty"`
	Disabled    bool     `json:"disabled"`
	Harnesses   []string `json:"harnesses,omitempty"`
	MCPServers         []string             `json:"mcpServers,omitempty"`
	MCPServerConfigs   []model.ADLMCPServer `json:"mcpServerConfigs,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	Rules            []string `json:"rules,omitempty"`
	AgentDeployers   []string `json:"agentDeployers,omitempty"`
	MentionProviders []string `json:"mentionProviders,omitempty"`
	HITLChannels     []string `json:"hitlChannels,omitempty"`
	StorageHandlers  []string `json:"storageHandlers,omitempty"`
	Agents        []string `json:"agents,omitempty"`
}

// Info returns API metadata for all extensions.
func (r *Registry) Info() []ExtensionInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	disabled := disabledExtensions()
	var out []ExtensionInfo
	for name, ext := range r.extensions {
		info := ExtensionInfo{
			Name:        name,
			Version:     ext.Manifest.Version,
			DisplayName: ext.Manifest.DisplayName,
			Description: ext.Manifest.Description,
			Disabled:    disabled[name],
		}
		for _, h := range ext.Harnesses {
			info.Harnesses = append(info.Harnesses, h.ID)
		}
		for _, s := range ext.MCPServers {
			info.MCPServers = append(info.MCPServers, s.Name)
			info.MCPServerConfigs = append(info.MCPServerConfigs, s)
		}
		for _, s := range ext.CustomMCPServers {
			info.MCPServers = append(info.MCPServers, s.Name)
		}
		for _, s := range ext.Skills {
			info.Skills = append(info.Skills, s.Name)
		}
		for _, s := range ext.CustomSkills {
			info.Skills = append(info.Skills, s.Name)
		}
		for _, rule := range ext.CustomRules {
			info.Rules = append(info.Rules, rule.Name)
		}
		for _, d := range ext.AgentDeployers {
			info.AgentDeployers = append(info.AgentDeployers, d.Name)
		}
		for _, mp := range ext.MentionProviders {
			info.MentionProviders = append(info.MentionProviders, mp.ID)
		}
		for _, ch := range ext.HITLChannels {
			info.HITLChannels = append(info.HITLChannels, ch.ID)
		}
		for _, sh := range ext.StorageHandlers {
			info.StorageHandlers = append(info.StorageHandlers, sh.ID)
		}
		for _, a := range ext.Agents {
			id := a.ID
			if strings.HasPrefix(id, extPrefix) {
				if _, item, ok := ParseExtRef(id); ok {
					id = item
				}
			}
			info.Agents = append(info.Agents, id)
		}
		out = append(out, info)
	}
	return out
}
