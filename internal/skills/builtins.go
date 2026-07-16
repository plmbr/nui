// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"loop/internal/model"
)

//go:embed builtins/*/SKILL.md
var builtinSkillFS embed.FS

const BuiltinRefPrefix = "builtin:"

var builtinSkillNames []string

func init() {
	names, err := listEmbeddedBuiltinSkillNames()
	if err != nil {
		panic(fmt.Sprintf("skills: load builtin skills: %v", err))
	}
	builtinSkillNames = names
}

func listEmbeddedBuiltinSkillNames() ([]string, error) {
	entries, err := fs.ReadDir(builtinSkillFS, "builtins")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, err := fs.Stat(builtinSkillFS, path.Join("builtins", name, skillFileName)); err != nil {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// IsBuiltinRef reports whether ref points at a compiled-in Loop skill.
func IsBuiltinRef(ref string) bool {
	return strings.HasPrefix(strings.TrimSpace(ref), BuiltinRefPrefix)
}

// IsBuiltinSkill reports whether name matches a compiled-in Loop skill.
func IsBuiltinSkill(name string) bool {
	name = strings.TrimSpace(name)
	for _, builtin := range builtinSkillNames {
		if builtin == name {
			return true
		}
	}
	return false
}

// BuiltinSkillNames returns compiled-in skill names shipped with Loop.
func BuiltinSkillNames() []string {
	out := make([]string, len(builtinSkillNames))
	copy(out, builtinSkillNames)
	return out
}

// BuiltinADLSkills returns ADL entries for all compiled-in skills.
func BuiltinADLSkills() []model.ADLSkill {
	names := BuiltinSkillNames()
	out := make([]model.ADLSkill, 0, len(names))
	for _, name := range names {
		out = append(out, model.ADLSkill{
			Name: name,
			Ref:  BuiltinRefPrefix + name,
		})
	}
	return out
}

// WithBuiltins appends compiled-in skills not already present by name.
// The ask-user and remember skills are attached separately when needed.
func WithBuiltins(skills []model.ADLSkill) []model.ADLSkill {
	if len(builtinSkillNames) == 0 {
		return skills
	}
	seen := make(map[string]bool, len(skills)+len(builtinSkillNames))
	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		if name != "" {
			seen[name] = true
		}
	}
	out := append([]model.ADLSkill{}, skills...)
	for _, builtin := range BuiltinADLSkills() {
		if builtin.Name == HitlAskUserSkillName || builtin.Name == RememberSkillName {
			continue
		}
		if seen[builtin.Name] {
			continue
		}
		out = append(out, builtin)
	}
	return out
}

const HitlAskUserSkillName = "ask-user"
const RememberSkillName = "remember"

// HitlAskUserSkill returns the compiled-in HITL ask-user skill reference.
func HitlAskUserSkill() model.ADLSkill {
	return model.ADLSkill{
		Name: HitlAskUserSkillName,
		Ref:  BuiltinRefPrefix + HitlAskUserSkillName,
	}
}

// RememberSkill returns the compiled-in memory remember skill reference.
func RememberSkill() model.ADLSkill {
	return model.ADLSkill{
		Name: RememberSkillName,
		Ref:  BuiltinRefPrefix + RememberSkillName,
	}
}

// ResolveBuiltin materializes a compiled-in skill into the catalog cache.
func ResolveBuiltin(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("builtin skill name is required")
	}
	if !IsBuiltinSkill(name) {
		return "", fmt.Errorf("builtin skill %q not found", name)
	}

	skillPath := path.Join("builtins", name, skillFileName)
	data, err := fs.ReadFile(builtinSkillFS, skillPath)
	if err != nil {
		return "", fmt.Errorf("builtin skill %q: %w", name, err)
	}

	dir, err := cacheSkillDir(builtinCacheName(name))
	if err != nil {
		return "", err
	}
	if err := writeSkillContent(dir, string(data)); err != nil {
		return "", err
	}
	if err := validateSkillDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// BuiltinBody returns the markdown body of a compiled-in skill without frontmatter.
func BuiltinBody(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("builtin skill name is required")
	}
	if !IsBuiltinSkill(name) {
		return "", fmt.Errorf("builtin skill %q not found", name)
	}
	skillPath := path.Join("builtins", name, skillFileName)
	data, err := fs.ReadFile(builtinSkillFS, skillPath)
	if err != nil {
		return "", fmt.Errorf("builtin skill %q: %w", name, err)
	}
	return strings.TrimSpace(stripFrontmatter(string(data))), nil
}

func builtinCacheName(name string) string {
	return "__builtin__" + name
}
