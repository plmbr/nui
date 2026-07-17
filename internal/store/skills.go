// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
)

// SkillsDir returns ~/.nui/skills, creating it if needed.
func SkillsDir() (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "skills")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func skillEntryDir(name string) (string, error) {
	base, err := SkillsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name), nil
}

func skillCacheDir(name string) (string, error) {
	entry, err := skillEntryDir(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(entry, "skill"), nil
}

func skillRepoDir(name string) (string, error) {
	entry, err := skillEntryDir(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(entry, "repo"), nil
}

// SkillEntryDir returns ~/.nui/skills/<name>.
func SkillEntryDir(name string) (string, error) {
	return skillEntryDir(name)
}

// SkillCacheDir returns ~/.nui/skills/<name>/skill.
func SkillCacheDir(name string) (string, error) {
	return skillCacheDir(name)
}

// SkillRepoDir returns ~/.nui/skills/<name>/repo.
func SkillRepoDir(name string) (string, error) {
	return skillRepoDir(name)
}
