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

	Harnesses  []HarnessEntry
	MCPServers []model.ADLMCPServer
	Skills     []model.ADLSkill
	Agents     []model.ADLDefinition

	defaultRuntime *RuntimeConfig
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
	mu          sync.RWMutex
	extensions  map[string]*Extension // name → extension
	catalogs    []*catalogProvider
	loadErrors  []string
}

// Default is the process-wide extension registry, set at server startup.
var Default *Registry

// LoadRegistry scans ~/.loop/extensions/*/extension.yaml and resolves contribution lists.
func LoadRegistry() (*Registry, error) {
	dir, err := store.ExtensionsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	reg := &Registry{extensions: map[string]*Extension{}}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		extDir := filepath.Join(dir, e.Name())
		ext, err := loadExtension(extDir, reg)
		if err != nil {
			reg.loadErrors = append(reg.loadErrors, fmt.Sprintf("%s: %v", e.Name(), err))
			fmt.Fprintf(os.Stderr, "[extensions] skip %s: %v\n", e.Name(), err)
			continue
		}
		reg.extensions[ext.Manifest.Name] = ext
	}
	Default = reg
	return reg, nil
}

func loadExtension(extDir string, reg *Registry) (*Extension, error) {
	manifest, err := LoadManifest(extDir)
	if err != nil {
		return nil, err
	}
	ext := &Extension{
		Dir:      extDir,
		Manifest: manifest,
	}
	if manifest.Contributions.Harnesses != nil && manifest.Contributions.Harnesses.Runtime != nil {
		rt := *manifest.Contributions.Harnesses.Runtime
		ext.defaultRuntime = &rt
	}

	var catalogCmd []string
	if manifest.Contributions.Catalog != nil {
		catalogCmd = manifest.Contributions.Catalog.Command
	}
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
		list, err := loadContributionList(extDir, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]HarnessEntry, error) { return loadHarnessesFromFile(file) },
			func(p *catalogProvider) ([]HarnessEntry, error) { return p.ListHarnesses() },
		)
		if err != nil {
			return nil, fmt.Errorf("harnesses: %w", err)
		}
		ext.Harnesses = list
	}

	if c := manifest.Contributions.MCPServers; c != nil {
		list, err := loadContributionList(extDir, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]model.ADLMCPServer, error) { return loadMCPServersFromFile(file) },
			func(p *catalogProvider) ([]model.ADLMCPServer, error) { return p.ListMCPServers() },
		)
		if err != nil {
			return nil, fmt.Errorf("mcpServers: %w", err)
		}
		ext.MCPServers = list
	}

	if c := manifest.Contributions.Skills; c != nil {
		list, err := loadContributionList(extDir, c.Source, catalogCmd, resolveCatalog,
			func(file string) ([]model.ADLSkill, error) { return loadSkillsFromFile(file) },
			func(p *catalogProvider) ([]model.ADLSkill, error) { return p.ListSkills() },
		)
		if err != nil {
			return nil, fmt.Errorf("skills: %w", err)
		}
		ext.Skills = list
	}

	if c := manifest.Contributions.Agents; c != nil {
		list, err := loadContributionList(extDir, c.Source, catalogCmd, resolveCatalog,
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

	return ext, nil
}

func loadContributionList[T any](
	extDir string,
	source Source,
	catalogCmd []string,
	resolveCatalog func() (*catalogProvider, error),
	fromFile func(string) ([]T, error),
	fromCatalog func(*catalogProvider) ([]T, error),
) ([]T, error) {
	file, _, err := source.resolved(extDir, catalogCmd)
	if err != nil {
		return nil, err
	}
	if file != "" {
		return fromFile(file)
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
	r.Shutdown()
	dir, err := store.ExtensionsDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	next := &Registry{extensions: map[string]*Extension{}}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		extDir := filepath.Join(dir, e.Name())
		ext, err := loadExtension(extDir, next)
		if err != nil {
			next.loadErrors = append(next.loadErrors, fmt.Sprintf("%s: %v", e.Name(), err))
			fmt.Fprintf(os.Stderr, "[extensions] skip %s: %v\n", e.Name(), err)
			continue
		}
		next.extensions[ext.Manifest.Name] = ext
	}
	r.mu.Lock()
	r.extensions = next.extensions
	r.catalogs = next.catalogs
	r.loadErrors = next.loadErrors
	r.mu.Unlock()
	Default = r
	return nil
}

// Shutdown stops catalog provider processes.
func (r *Registry) Shutdown() {
	r.mu.Lock()
	catalogs := r.catalogs
	r.mu.Unlock()
	for _, c := range catalogs {
		_ = c.Close()
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
	if h.Runtime != nil {
		return *h.Runtime, true
	}
	if ext.defaultRuntime != nil {
		return *ext.defaultRuntime, true
	}
	return RuntimeConfig{}, false
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
	MCPServers  []string `json:"mcpServers,omitempty"`
	Skills      []string `json:"skills,omitempty"`
	Agents      []string `json:"agents,omitempty"`
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
		}
		for _, s := range ext.Skills {
			info.Skills = append(info.Skills, s.Name)
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
