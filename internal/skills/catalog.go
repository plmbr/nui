// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nui/internal/extensions"
	"nui/internal/model"
	"nui/internal/store"
)

// Entry describes a skill installed in the nui catalog.
type Entry struct {
	Name        string `json:"name"`
	Source      string `json:"source"` // local | ref | content | git
	Path        string `json:"path,omitempty"`
	Git         string `json:"git,omitempty"`
	Version     string `json:"version,omitempty"`
	InstalledAt string `json:"installedAt"`
}

type manifest struct {
	Skills map[string]Entry `json:"skills"`
}

func loadManifest() (manifest, error) {
	dir, err := store.SkillsDir()
	if err != nil {
		return manifest{}, err
	}
	path := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return manifest{Skills: map[string]Entry{}}, nil
	}
	if err != nil {
		return manifest{}, err
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return manifest{}, err
	}
	if m.Skills == nil {
		m.Skills = map[string]Entry{}
	}
	return m, nil
}

func saveManifest(m manifest) error {
	dir, err := store.SkillsDir()
	if err != nil {
		return err
	}
	if m.Skills == nil {
		m.Skills = map[string]Entry{}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644)
}

func cacheSkillDir(name string) (string, error) {
	return store.SkillCacheDir(name)
}

// List returns skills from the catalog manifest plus any skill directories under
// ~/.nui/skills/<name>/ that contain SKILL.md (with or without a manifest entry).
func List() ([]Entry, error) {
	m, err := loadManifest()
	if err != nil {
		return nil, err
	}
	byName := make(map[string]Entry, len(m.Skills))
	for name, e := range m.Skills {
		byName[name] = e
	}
	dir, err := store.SkillsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, ok := byName[name]; ok {
			continue
		}
		skillDir, ok := skillDirInEntry(filepath.Join(dir, name))
		if !ok {
			continue
		}
		byName[name] = Entry{
			Name:   name,
			Source: "local",
			Path:   skillDir,
		}
	}
	out := make([]Entry, 0, len(byName))
	for _, e := range byName {
		out = append(out, e)
	}
	return out, nil
}

func recordEntry(e Entry) error {
	m, err := loadManifest()
	if err != nil {
		return err
	}
	if e.InstalledAt == "" {
		e.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	}
	m.Skills[e.Name] = e
	return saveManifest(m)
}

func installed(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	m, err := loadManifest()
	if err == nil {
		if _, ok := m.Skills[name]; ok {
			return true
		}
	}
	entryDir, err := store.SkillEntryDir(name)
	if err != nil {
		return false
	}
	if _, ok := skillDirInEntry(entryDir); ok {
		return true
	}
	return false
}

// InstallLocal copies a local skill directory into ~/.nui/skills/<name>/skill/.
func InstallLocal(name, srcPath string, overwrite bool) (string, error) {
	src, err := localSkillDir(srcPath)
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = filepath.Base(src)
	}
	if name == "" {
		return "", fmt.Errorf("skill name is required")
	}
	if installed(name) && !overwrite {
		return "", fmt.Errorf("already exists: %q", name)
	}
	dst, err := cacheSkillDir(name)
	if err != nil {
		return "", err
	}
	if err := replaceDirContents(src, dst); err != nil {
		return "", err
	}
	if err := recordEntry(Entry{
		Name:   name,
		Source: "local",
		Path:   srcPath,
	}); err != nil {
		return "", err
	}
	return name, nil
}

// InstallContent writes inline SKILL.md content into the catalog.
func InstallContent(name, content string, overwrite bool) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("skill content is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		var err error
		name, err = defaultNameFromContent(content)
		if err != nil {
			return "", err
		}
	}
	if installed(name) && !overwrite {
		return "", fmt.Errorf("already exists: %q", name)
	}
	dst, err := cacheSkillDir(name)
	if err != nil {
		return "", err
	}
	if err := writeSkillContent(dst, content); err != nil {
		return "", err
	}
	if err := recordEntry(Entry{
		Name:   name,
		Source: "content",
	}); err != nil {
		return "", err
	}
	return name, nil
}

// InstallGit clones a repo and copies the skill subdirectory into the catalog.
func InstallGit(name, gitURL, repoPath, version string, overwrite bool) (string, error) {
	gitURL = strings.TrimSpace(gitURL)
	repoPath = strings.TrimSpace(repoPath)
	if gitURL == "" {
		return "", fmt.Errorf("git url is required")
	}
	repoPath = normalizeRepoSkillPath(repoPath)
	if repoPath == "" {
		return "", fmt.Errorf("path (relative skill directory or SKILL.md in repo) is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		var err error
		name, err = DefaultSkillNameFromPath(repoPath)
		if err != nil {
			name = repoNameFromGitURL(gitURL)
		}
	}
	if name == "" {
		return "", fmt.Errorf("skill name is required")
	}
	if installed(name) && !overwrite {
		return "", fmt.Errorf("already exists: %q", name)
	}

	skillDir, err := ensureGitSkill(name, gitURL, repoPath, version)
	if err != nil {
		return "", err
	}
	dst, err := cacheSkillDir(name)
	if err != nil {
		return "", err
	}
	if err := replaceDirContents(skillDir, dst); err != nil {
		return "", err
	}
	if err := recordEntry(Entry{
		Name:    name,
		Source:  "git",
		Git:     gitURL,
		Path:    repoPath,
		Version: version,
	}); err != nil {
		return "", err
	}
	return name, nil
}

// Remove deletes a skill from the catalog.
func Remove(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("skill name is required")
	}
	entryDir, err := store.SkillEntryDir(name)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(entryDir); err != nil {
		return err
	}
	m, err := loadManifest()
	if err != nil {
		return err
	}
	delete(m.Skills, name)
	return saveManifest(m)
}

// Resolve returns a local directory containing SKILL.md for an ADL skill entry.
func Resolve(ctx Context, skill model.ADLSkill) (string, error) {
	if err := model.ValidateADLSkills([]model.ADLSkill{skill}); err != nil {
		return "", err
	}
	kind, err := model.SkillSourceKind(skill)
	if err != nil {
		return "", err
	}

	switch kind {
	case "local":
		return localSkillDir(skill.Path)
	case "ref":
		return resolveRef(ctx, strings.TrimSpace(skill.Ref))
	case "content":
		return resolveContent(skill.Name, skill.Content)
	case "git":
		return ensureGitSkill(skill.Name, skill.Git, skill.Path, skill.Version)
	default:
		return "", fmt.Errorf("skill %q: unsupported source", skill.Name)
	}
}

func resolveRef(ctx Context, ref string) (string, error) {
	if IsBuiltinRef(ref) {
		return ResolveBuiltin(strings.TrimPrefix(strings.TrimSpace(ref), BuiltinRefPrefix))
	}
	if extensions.IsExtRef(ref) && extensions.Default != nil {
		_, dir, err := extensions.Default.ResolveSkill(ref)
		if err == nil {
			if err := validateSkillDir(dir); err == nil {
				return dir, nil
			}
			return "", err
		}
	}
	for _, p := range refSearchPaths(ctx, ref) {
		if err := validateSkillDir(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("skill ref %q not found in catalog or .cursor/skills", ref)
}

func resolveContent(name, content string) (string, error) {
	if cache, err := cacheSkillDir(name); err == nil {
		if err := validateSkillDir(cache); err == nil {
			return cache, nil
		}
	}
	dir, err := cacheSkillDir(name)
	if err != nil {
		return "", err
	}
	if err := writeSkillContent(dir, content); err != nil {
		return "", err
	}
	return dir, nil
}
