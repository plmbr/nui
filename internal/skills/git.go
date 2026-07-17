// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nui/internal/store"
)

func ensureGitSkill(name, gitURL, repoPath, version string) (string, error) {
	repoDir, err := store.SkillRepoDir(name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(repoDir), 0700); err != nil {
		return "", err
	}
	if err := cloneOrUpdateRepo(repoDir, gitURL, version); err != nil {
		return "", err
	}
	repoPath = normalizeRepoSkillPath(repoPath)
	skillDir := filepath.Join(repoDir, filepath.FromSlash(repoPath))
	if err := validateSkillDir(skillDir); err != nil {
		return "", fmt.Errorf("git skill %q at %q: %w", name, repoPath, err)
	}
	return skillDir, nil
}

func cloneOrUpdateRepo(repoDir, gitURL, version string) error {
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		return gitClone(repoDir, gitURL, version)
	}
	if version != "" {
		return gitCheckout(repoDir, version)
	}
	cmd := exec.Command("git", "-C", repoDir, "fetch", "--depth", "1", "origin")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch: %w", err)
	}
	return nil
}

func gitClone(repoDir, gitURL, version string) error {
	parent := filepath.Dir(repoDir)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	_ = os.RemoveAll(repoDir)

	args := []string{"clone", "--depth", "1"}
	if version != "" {
		args = append(args, "--branch", version)
	}
	args = append(args, gitURL, repoDir)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	if version != "" {
		return gitCheckout(repoDir, version)
	}
	return nil
}

func gitCheckout(repoDir, version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil
	}
	cmd := exec.Command("git", "-C", repoDir, "checkout", version)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout %q: %w", version, err)
	}
	return nil
}
