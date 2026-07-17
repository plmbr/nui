// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nui/internal/store"
)

const skillFileName = "SKILL.md"

// Context carries session-local paths used when resolving skill refs.
type Context struct {
	WorkingDir string
}

// ExpandPath expands ~ and returns an absolute path.
func ExpandPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	if p[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if len(p) == 1 || p[1] == '/' || p[1] == '\\' {
			p = filepath.Join(home, p[1:])
		}
	}
	return filepath.Abs(p)
}

func validateSkillDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("skill directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("skill path %q is not a directory", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, skillFileName)); err != nil {
		return fmt.Errorf("skill directory %q: missing %s", dir, skillFileName)
	}
	return nil
}

func localSkillDir(path string) (string, error) {
	abs, err := ExpandPath(path)
	if err != nil {
		return "", err
	}

	switch {
	case strings.HasSuffix(abs, skillFileName):
		dir := filepath.Dir(abs)
		if err := validateSkillDir(dir); err != nil {
			return "", err
		}
		return dir, nil
	default:
		if err := validateSkillDir(abs); err != nil {
			return "", err
		}
		return abs, nil
	}
}

func refSearchPaths(ctx Context, ref string) []string {
	var paths []string
	if entry, err := store.SkillEntryDir(ref); err == nil {
		paths = append(paths, entry)
	}
	if cache, err := cacheSkillDir(ref); err == nil {
		paths = append(paths, cache)
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".cursor", "skills", ref))
	}
	if wd := strings.TrimSpace(ctx.WorkingDir); wd != "" {
		paths = append(paths, filepath.Join(wd, ".cursor", "skills", ref))
	}
	return paths
}

// skillDirInEntry returns the directory containing SKILL.md under ~/.nui/skills/<name>/.
// Supports both ~/.nui/skills/<name>/SKILL.md and ~/.nui/skills/<name>/skill/SKILL.md.
func skillDirInEntry(entryDir string) (string, bool) {
	if err := validateSkillDir(entryDir); err == nil {
		return entryDir, true
	}
	cache := filepath.Join(entryDir, "skill")
	if err := validateSkillDir(cache); err == nil {
		return cache, true
	}
	return "", false
}
