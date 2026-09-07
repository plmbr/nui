// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"nui/internal/model"
)

// MetadataAppendix returns a compact skill catalog (name + description) for API prompts.
// Full skill bodies are loaded on demand via the nui-skills MCP server.
func MetadataAppendix(ctx Context, skillList []model.ADLSkill) string {
	skillList = WithBuiltins(skillList)
	var lines []string
	for _, skill := range skillList {
		name := strings.TrimSpace(skill.Name)
		if name == "" || name == HitlAskUserSkillName {
			continue
		}
		desc, err := ReadDescription(ctx, skill)
		if err != nil {
			desc = ""
		}
		if desc == "" {
			lines = append(lines, fmt.Sprintf("- **%s**", name))
			continue
		}
		lines = append(lines, fmt.Sprintf("- **%s**: %s", name, desc))
	}
	if len(lines) == 0 {
		return ""
	}
	return "## Available skills\n\n" +
		"Use `load_skill` on the **nui-skills** MCP server to load full instructions when a skill is relevant. " +
		"Then use **nui-fs** / **nui-bash** for skill files and scripts under the returned skill root.\n\n" +
		strings.Join(lines, "\n")
}

// ReadDescription returns the YAML frontmatter description for a skill.
func ReadDescription(ctx Context, skill model.ADLSkill) (string, error) {
	raw, err := readSkillMarkdown(ctx, skill)
	if err != nil {
		return "", err
	}
	return frontmatterField(raw, "description"), nil
}

func readSkillMarkdown(ctx Context, skill model.ADLSkill) (string, error) {
	name := strings.TrimSpace(skill.Name)
	if IsBuiltinRef(skill.Ref) || (skill.Ref == "" && IsBuiltinSkill(name)) {
		data, err := builtinSkillFS.ReadFile(path.Join("builtins", name, skillFileName))
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	dir, err := Resolve(ctx, skill)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func frontmatterField(content, key string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	rest := strings.TrimPrefix(content, "---")
	end := strings.Index(rest, "---")
	if end < 0 {
		return ""
	}
	prefix := key + ":"
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		return strings.Trim(val, `"'`)
	}
	return ""
}
