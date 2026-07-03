// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"loop/internal/model"
)

// ExpandSlashCommand replaces a leading /skill-name invocation with the skill body.
// Harnesses running in headless/stream mode do not expand slash commands themselves.
func ExpandSlashCommand(ctx Context, available []model.ADLSkill, message string) (string, error) {
	name, args, ok := parseSlashSkillInvocation(message)
	if !ok {
		return message, nil
	}

	var skill *model.ADLSkill
	for i := range available {
		if available[i].Name == name {
			skill = &available[i]
			break
		}
	}
	if skill == nil {
		return message, nil
	}

	dir, err := Resolve(ctx, *skill)
	if err != nil {
		return "", fmt.Errorf("skill %q: %w", name, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, skillFileName))
	if err != nil {
		return "", fmt.Errorf("skill %q: read %s: %w", name, skillFileName, err)
	}

	expanded := strings.TrimSpace(stripFrontmatter(string(data)))
	if args != "" {
		if expanded != "" {
			expanded += "\n\n"
		}
		expanded += strings.TrimSpace(args)
	}
	if expanded == "" {
		return message, nil
	}
	return expanded, nil
}

func parseSlashSkillInvocation(message string) (name, args string, ok bool) {
	message = strings.TrimSpace(message)
	if !strings.HasPrefix(message, "/") {
		return "", "", false
	}
	rest := message[1:]
	end := 0
	for end < len(rest) && isSkillNameChar(rest[end]) {
		end++
	}
	if end == 0 {
		return "", "", false
	}
	name = rest[:end]
	args = strings.TrimSpace(rest[end:])
	return name, args, true
}

func isSkillNameChar(b byte) bool {
	return unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b)) || b == '-' || b == '_'
}

func stripFrontmatter(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return content
	}
	rest := strings.TrimPrefix(content, "---")
	end := strings.Index(rest, "---")
	if end < 0 {
		return content
	}
	return strings.TrimSpace(rest[end+3:])
}
