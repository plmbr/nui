// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"loop/internal/model"
)

// MaterializeSkill resolves and copies a skill into destDir (replacing existing contents).
func MaterializeSkill(ctx Context, skill model.ADLSkill, destDir string) error {
	src, err := Resolve(ctx, skill)
	if err != nil {
		return fmt.Errorf("skill %q: %w", skill.Name, err)
	}
	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	return CopyDir(src, destDir)
}

// MaterializeSkills resolves and copies all skills into a harness session config directory.
func MaterializeSkills(ctx Context, harnessType, configDir string, skills []model.ADLSkill) error {
	if err := model.ValidateADLSkills(skills); err != nil {
		return err
	}
	for _, skill := range skills {
		dest := HarnessSkillDir(harnessType, configDir, skill.Name)
		if err := MaterializeSkill(ctx, skill, dest); err != nil {
			return err
		}
	}
	return nil
}

// HarnessSkillDir returns the harness-specific install path for one skill.
func HarnessSkillDir(harnessType, configDir, skillName string) string {
	switch normalizeHarnessType(harnessType) {
	case "claude-code", "":
		// CLAUDE_CONFIG_DIR points at the session config root (mirrors ~/.claude).
		// Skills belong at $CLAUDE_CONFIG_DIR/skills/, not .claude/skills/.
		return filepath.Join(configDir, "skills", skillName)
	case "pi":
		return filepath.Join(configDir, piAgentSubdir, "skills", skillName)
	default:
		return filepath.Join(configDir, "skills", skillName)
	}
}

const piAgentSubdir = "pi-agent"

func normalizeHarnessType(harnessType string) string {
	if harnessType == "" {
		return "claude-code"
	}
	return harnessType
}
