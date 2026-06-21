// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NormalizeADLSkills maps legacy top-level skill into aiAssets.skills when needed.
func NormalizeADLSkills(def *ADLDefinition) {
	if def == nil {
		return
	}
	if strings.TrimSpace(def.Skill) == "" || len(def.AIAssets.Skills) > 0 {
		return
	}
	name := skillNameFromPath(def.Skill)
	def.AIAssets.Skills = []ADLSkill{{
		Name: name,
		Path: def.Skill,
	}}
}

func skillNameFromPath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasSuffix(p, "SKILL.md") {
		return filepath.Base(filepath.Dir(p))
	}
	return filepath.Base(p)
}

// ValidateADLSkills checks skill entries for required fields and a single source type.
func ValidateADLSkills(skills []ADLSkill) error {
	seen := map[string]bool{}
	for _, s := range skills {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			return fmt.Errorf("skill name is required")
		}
		if seen[name] {
			return fmt.Errorf("duplicate skill name %q", name)
		}
		seen[name] = true
		if _, err := SkillSourceKind(s); err != nil {
			return err
		}
	}
	return nil
}

// SkillSourceKind returns local, ref, content, or git for a skill entry.
func SkillSourceKind(s ADLSkill) (string, error) {
	hasPath := strings.TrimSpace(s.Path) != ""
	hasRef := strings.TrimSpace(s.Ref) != ""
	hasGit := strings.TrimSpace(s.Git) != ""
	hasContent := strings.TrimSpace(s.Content) != ""

	switch {
	case hasContent && !hasPath && !hasRef && !hasGit:
		return "content", nil
	case hasRef && !hasPath && !hasGit && !hasContent:
		return "ref", nil
	case hasGit:
		if !hasPath {
			return "", fmt.Errorf("skill %q: git source requires path (relative skill directory in repo)", s.Name)
		}
		if hasRef || hasContent {
			return "", fmt.Errorf("skill %q: ambiguous source (git cannot combine with ref or content)", s.Name)
		}
		return "git", nil
	case hasPath && !hasGit && !hasRef && !hasContent:
		return "local", nil
	default:
		return "", fmt.Errorf("skill %q: requires exactly one source (path, ref, content, or git+path)", s.Name)
	}
}
