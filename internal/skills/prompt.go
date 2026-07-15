// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"loop/internal/model"
)

// PromptAppendix returns skill bodies for inlining into an api harness system prompt.
func PromptAppendix(ctx Context, skillList []model.ADLSkill) string {
	skillList = WithBuiltins(skillList)
	var blocks []string
	for _, skill := range skillList {
		if skill.Name == HitlAskUserSkillName {
			continue
		}
		body, err := ReadBody(ctx, skill)
		if err != nil {
			continue
		}
		blocks = append(blocks, fmt.Sprintf(
			"### Skill: %s\n\nInvoke with `/%s` or follow these instructions when relevant:\n\n%s",
			skill.Name,
			skill.Name,
			body,
		))
	}
	if len(blocks) == 0 {
		return ""
	}
	return "## Loop skills\n\n" + strings.Join(blocks, "\n\n")
}

// ReadBody resolves a skill and returns its markdown body without YAML frontmatter.
func ReadBody(ctx Context, skill model.ADLSkill) (string, error) {
	name := strings.TrimSpace(skill.Name)
	if IsBuiltinRef(skill.Ref) || (skill.Ref == "" && IsBuiltinSkill(name)) {
		return BuiltinBody(name)
	}
	dir, err := Resolve(ctx, skill)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(StripFrontmatter(string(data))), nil
}

// StripFrontmatter removes optional YAML frontmatter from skill markdown.
func StripFrontmatter(content string) string {
	return stripFrontmatter(content)
}
