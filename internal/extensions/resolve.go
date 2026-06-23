// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"loop/internal/model"
	"loop/internal/store"
)

// ResolveMCPRef resolves ext:<extension>/<server-name> to an ADLMCPServer.
func (r *Registry) ResolveMCPRef(ref string) (model.ADLMCPServer, error) {
	extName, serverName, ok := ParseExtRef(ref)
	if !ok {
		return model.ADLMCPServer{}, fmt.Errorf("invalid mcp ref %q", ref)
	}
	ext, ok := r.Get(extName)
	if !ok || r.isDisabled(extName) {
		return model.ADLMCPServer{}, fmt.Errorf("extension %q not found", extName)
	}
	for _, s := range ext.MCPServers {
		if s.Name == serverName {
			out := s
			if out.Name == "" {
				out.Name = serverName
			}
			return out, nil
		}
	}
	return model.ADLMCPServer{}, fmt.Errorf("mcp server %q not found in extension %q", serverName, extName)
}

// ResolveSkill returns skill metadata and directory path for ext:<extension>/<skill-name>.
func (r *Registry) ResolveSkill(ref string) (model.ADLSkill, string, error) {
	extName, skillName, ok := ParseExtRef(ref)
	if !ok {
		return model.ADLSkill{}, "", fmt.Errorf("invalid skill ref %q", ref)
	}
	ext, ok := r.Get(extName)
	if !ok || r.isDisabled(extName) {
		return model.ADLSkill{}, "", fmt.Errorf("extension %q not found", extName)
	}
	for _, s := range ext.Skills {
		if s.Name != skillName {
			continue
		}
		dir, err := skillDir(ext.Dir, s)
		if err != nil {
			return model.ADLSkill{}, "", err
		}
		return s, dir, nil
	}
	return model.ADLSkill{}, "", fmt.Errorf("skill %q not found in extension %q", skillName, extName)
}

func skillDir(extDir string, skill model.ADLSkill) (string, error) {
	kind, err := model.SkillSourceKind(skill)
	if err != nil {
		return "", err
	}
	switch kind {
	case "local":
		p := strings.TrimSpace(skill.Path)
		if p == "" {
			return "", fmt.Errorf("skill %q: path is required", skill.Name)
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(extDir, p)
		}
		return p, nil
	case "content":
		cacheName := "ext-" + filepath.Base(extDir) + "-" + skill.Name
		dir, err := store.SkillCacheDir(cacheName)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill.Content), 0644); err != nil {
			return "", err
		}
		return dir, nil
	default:
		return "", fmt.Errorf("skill %q: extension skills support path or content only", skill.Name)
	}
}

// FindAgent looks up an extension-contributed agent by global id.
func (r *Registry) FindAgent(agentID string) (model.ADLDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for name, ext := range r.extensions {
		if r.isDisabled(name) {
			continue
		}
		for _, a := range ext.Agents {
			if a.ID == agentID || model.ADLAgentID(a) == agentID {
				return a, true
			}
		}
	}
	// Also match ext:<ext>/<local-id>
	extName, localID, ok := ParseExtRef(agentID)
	if !ok {
		return model.ADLDefinition{}, false
	}
	ext, ok := r.extensions[extName]
	if !ok || r.isDisabled(extName) {
		return model.ADLDefinition{}, false
	}
	for _, a := range ext.Agents {
		id := a.ID
		if strings.HasPrefix(id, extPrefix) {
			if _, item, ok := ParseExtRef(id); ok {
				if item == localID {
					return a, true
				}
			}
		}
	}
	return model.ADLDefinition{}, false
}

// ExpandMCPServers resolves ref-only MCP entries in place.
func (r *Registry) ExpandMCPServers(servers []model.ADLMCPServer) ([]model.ADLMCPServer, error) {
	if len(servers) == 0 {
		return servers, nil
	}
	out := make([]model.ADLMCPServer, 0, len(servers))
	for _, s := range servers {
		ref := strings.TrimSpace(s.Ref)
		if ref == "" {
			out = append(out, s)
			continue
		}
		resolved, err := r.ResolveMCPRef(ref)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(s.Name) != "" {
			resolved.Name = s.Name
		}
		out = append(out, resolved)
	}
	return out, nil
}

// LoopMCPServerConfigs returns MCP server configs from all extensions for Loop-side MCP manager.
func (r *Registry) LoopMCPServerConfigs() map[string]map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string]map[string]any{}
	for extName, ext := range r.extensions {
		if r.isDisabled(extName) {
			continue
		}
		for _, s := range ext.MCPServers {
			name := fmt.Sprintf("ext-%s-%s", extName, s.Name)
			entry := map[string]any{}
			if s.Command != "" {
				entry["command"] = s.Command
				if len(s.Args) > 0 {
					entry["args"] = s.Args
				}
			}
			if s.URL != "" {
				entry["url"] = s.URL
			}
			if len(entry) > 0 {
				out[name] = entry
			}
		}
	}
	return out
}
