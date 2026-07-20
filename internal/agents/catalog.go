// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nui/internal/model"
	"nui/internal/skills"
	"nui/internal/store"

	"gopkg.in/yaml.v3"
)

// Install copies an ADL agent YAML from a local path or git URL into ~/.nui/agents/.
// Returns the installed agent id.
func Install(source string) (string, error) {
	content, err := loadAgentYAML(source)
	if err != nil {
		return "", err
	}
	var def model.ADLDefinition
	if err := yaml.Unmarshal(content, &def); err != nil {
		return "", fmt.Errorf("parse agent ADL: %w", err)
	}
	model.NormalizeADLDefinition(&def)
	model.NormalizeADLSkills(&def)
	if err := model.ValidateADLDefinition(def); err != nil {
		return "", fmt.Errorf("invalid agent ADL: %w", err)
	}

	filename, err := agentFilename(def)
	if err != nil {
		return "", err
	}
	dir, err := store.AgentsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, content, 0644); err != nil {
		return "", err
	}
	return model.ADLAgentID(def), nil
}

// Remove deletes a user-installed agent by id or filename.
func Remove(idOrFile string) error {
	idOrFile = strings.TrimSpace(idOrFile)
	if idOrFile == "" {
		return fmt.Errorf("agent id or filename is required")
	}
	for _, def := range builtinAgentDefs {
		if model.ADLAgentID(def) == idOrFile {
			return fmt.Errorf("cannot remove builtin agent %q", idOrFile)
		}
	}
	if strings.HasPrefix(idOrFile, "ext:") {
		return fmt.Errorf("cannot remove extension agent %q; use nui extension remove", idOrFile)
	}

	dir, err := store.AgentsDir()
	if err != nil {
		return err
	}
	path, err := resolveAgentPath(dir, idOrFile)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("agent %q is not installed", idOrFile)
		}
		return err
	}
	return nil
}

func loadAgentYAML(source string) ([]byte, error) {
	source = normalizeSource(source)
	if source == "" {
		return nil, fmt.Errorf("source is required")
	}
	if skills.IsGitRemote(source) {
		return loadAgentYAMLFromGit(source)
	}
	return loadAgentYAMLFromLocal(source)
}

func loadAgentYAMLFromLocal(source string) ([]byte, error) {
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("source %q: %w", source, err)
	}
	if !info.IsDir() {
		if !isYAMLFile(source) {
			return nil, fmt.Errorf("source %q: expected a .yaml or .yml file", source)
		}
		return os.ReadFile(source)
	}
	matches, err := yamlFilesInDir(source)
	if err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no ADL yaml files found in %q", source)
	case 1:
		return os.ReadFile(matches[0])
	default:
		return nil, fmt.Errorf("multiple yaml files in %q; specify the agent file", source)
	}
}

func loadAgentYAMLFromGit(source string) ([]byte, error) {
	cloneURL, repoPath, ref, ok := skills.ParseGitHubURL(source)
	if !ok {
		if isGitURL(source) {
			cloneURL = source
		} else {
			return nil, fmt.Errorf("unsupported git url %q", source)
		}
	}
	tmp, err := os.MkdirTemp("", "nui-agent-clone-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, cloneURL, tmp)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git clone: %w", err)
	}

	repoPath = strings.Trim(strings.TrimSpace(repoPath), "/")
	if repoPath != "" {
		path := filepath.Join(tmp, filepath.FromSlash(repoPath))
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("git path %q: %w", repoPath, err)
		}
		if info.IsDir() {
			matches, err := yamlFilesInDir(path)
			if err != nil {
				return nil, err
			}
			switch len(matches) {
			case 0:
				return nil, fmt.Errorf("no ADL yaml files at %q", repoPath)
			case 1:
				return os.ReadFile(matches[0])
			default:
				return nil, fmt.Errorf("multiple yaml files at %q; use a blob/tree URL to a specific file", repoPath)
			}
		}
		if !isYAMLFile(path) {
			return nil, fmt.Errorf("git path %q is not a yaml file", repoPath)
		}
		return os.ReadFile(path)
	}

	matches, err := yamlFilesInDir(tmp)
	if err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no ADL yaml files in repository; use a GitHub tree/blob URL to the agent file")
	case 1:
		return os.ReadFile(matches[0])
	default:
		return nil, fmt.Errorf("multiple yaml files in repository root; use a tree/blob URL to the agent file")
	}
}

func agentFilename(def model.ADLDefinition) (string, error) {
	id := model.ADLAgentID(def)
	name := strings.TrimSpace(id)
	if name == "" {
		return "", fmt.Errorf("agent id is required")
	}
	if filepath.Base(name) != name || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid agent id %q", id)
	}
	ext := filepath.Ext(name)
	if ext != ".yaml" && ext != ".yml" {
		name += ".yaml"
	}
	return name, nil
}

func resolveAgentPath(dir, idOrFile string) (string, error) {
	candidates := []string{idOrFile}
	if !strings.HasSuffix(idOrFile, ".yaml") && !strings.HasSuffix(idOrFile, ".yml") {
		candidates = append(candidates, idOrFile+".yaml", idOrFile+".yml")
	}
	for _, name := range candidates {
		if filepath.Base(name) != name || strings.Contains(name, "..") {
			continue
		}
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	byID, err := agentFilesByID()
	if err != nil {
		return "", err
	}
	if file, ok := byID[idOrFile]; ok {
		return filepath.Join(dir, file), nil
	}
	return "", fmt.Errorf("agent %q is not installed", idOrFile)
}

func agentFilesByID() (map[string]string, error) {
	dir, err := store.AgentsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !isYAMLFile(e.Name()) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var def model.ADLDefinition
		if err := yaml.Unmarshal(raw, &def); err != nil {
			continue
		}
		model.NormalizeADLDefinition(&def)
		out[model.ADLAgentID(def)] = e.Name()
	}
	return out, nil
}

func yamlFilesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if isYAMLFile(e.Name()) {
			matches = append(matches, filepath.Join(dir, e.Name()))
		}
	}
	return matches, nil
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}

func normalizeSource(source string) string {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, source[2:])
		}
	}
	return source
}

func isGitURL(source string) bool {
	switch {
	case strings.HasPrefix(source, "http://"), strings.HasPrefix(source, "https://"):
		return true
	case strings.HasPrefix(source, "git@"):
		return true
	case strings.HasPrefix(source, "git://"), strings.HasPrefix(source, "ssh://"):
		return true
	default:
		return false
	}
}
