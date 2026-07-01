// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// DefaultSkillNameFromPath infers a catalog name from a local path or repo-relative path.
// Paths pointing at SKILL.md use the parent directory name.
func DefaultSkillNameFromPath(skillPath string) (string, error) {
	skillPath = strings.TrimSpace(skillPath)
	if skillPath == "" {
		return "", fmt.Errorf("cannot infer skill name from empty path")
	}
	slash := filepath.ToSlash(skillPath)
	slash = strings.Trim(slash, "/")
	if slash == "" || slash == "." {
		return "", fmt.Errorf("cannot infer skill name from %q", skillPath)
	}
	if strings.EqualFold(path.Base(slash), skillFileName) {
		slash = path.Dir(slash)
		slash = strings.Trim(slash, "/")
	}
	if slash == "" || slash == "." {
		return "", fmt.Errorf("cannot infer skill name from %q", skillPath)
	}
	return path.Base(slash), nil
}

func defaultNameFromContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return "", fmt.Errorf("skill name is required (use --name or add name to SKILL.md frontmatter)")
	}
	rest := strings.TrimPrefix(content, "---")
	end := strings.Index(rest, "---")
	if end < 0 {
		return "", fmt.Errorf("skill name is required (use --name or add name to SKILL.md frontmatter)")
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "name:") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		name = strings.Trim(name, `"'`)
		if name != "" {
			return name, nil
		}
	}
	return "", fmt.Errorf("skill name is required (use --name or add name to SKILL.md frontmatter)")
}

func repoNameFromGitURL(gitURL string) string {
	gitURL = strings.TrimSpace(gitURL)
	gitURL = strings.TrimSuffix(gitURL, ".git")
	if cloneURL, _, _, ok := ParseGitHubURL(gitURL); ok {
		gitURL = strings.TrimSuffix(cloneURL, ".git")
	}
	if i := strings.LastIndex(gitURL, "/"); i >= 0 && i < len(gitURL)-1 {
		return gitURL[i+1:]
	}
	return filepath.Base(gitURL)
}
