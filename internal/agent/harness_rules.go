// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"loop/internal/extensions"
	"loop/internal/model"
)

var ruleFilenameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// ResolvedRule is a harness rule with resolved body text ready to materialize.
type ResolvedRule struct {
	Name    string
	Content string
}

func resolveHarnessRules(rules []model.ADLRule, reg *extensions.Registry) ([]ResolvedRule, error) {
	if len(rules) == 0 {
		return nil, nil
	}
	out := make([]ResolvedRule, 0, len(rules))
	for _, rule := range rules {
		name := strings.TrimSpace(rule.Name)
		if name == "" {
			name = fmt.Sprintf("rule-%d", len(out)+1)
		}
		ref := strings.TrimSpace(rule.Ref)
		var body string
		switch {
		case ref != "" && extensions.IsExtRef(ref):
			if reg == nil {
				return nil, fmt.Errorf("rule ref %q requires extension registry", ref)
			}
			resolved, err := reg.ResolveRule(ref)
			if err != nil {
				return nil, err
			}
			body = resolved
		case strings.TrimSpace(rule.Content) != "":
			body = strings.TrimSpace(rule.Content)
		case strings.TrimSpace(rule.Path) != "":
			data, err := os.ReadFile(rule.Path)
			if err != nil {
				return nil, err
			}
			body = string(data)
		default:
			continue
		}
		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}
		out = append(out, ResolvedRule{Name: name, Content: body})
	}
	return out, nil
}

func ruleFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "rule.md"
	}
	safe := strings.Trim(ruleFilenameSanitizer.ReplaceAllString(name, "-"), "-")
	if safe == "" {
		return "rule.md"
	}
	if !strings.HasSuffix(strings.ToLower(safe), ".md") {
		safe += ".md"
	}
	return safe
}

// harnessRulesRoot returns the directory where rule files are written for a harness.
func harnessRulesRoot(harnessType, configDir string) string {
	switch normalizeHarnessType(harnessType) {
	case "pi":
		return filepath.Join(piAgentConfigDir(configDir), "rules")
	default:
		return filepath.Join(configDir, "rules")
	}
}

// installHarnessRules writes resolved rules as markdown files for the harness type.
// Returns paths relative to configDir (harness config root).
func installHarnessRules(harnessType, configDir string, rules []ResolvedRule) ([]string, error) {
	if len(rules) == 0 {
		return nil, nil
	}
	rulesDir := harnessRulesRoot(harnessType, configDir)
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return nil, err
	}
	var relPaths []string
	for _, rule := range rules {
		filename := ruleFilename(rule.Name)
		absPath := filepath.Join(rulesDir, filename)
		content := strings.TrimSpace(rule.Content) + "\n"
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(configDir, absPath)
		if err != nil {
			rel = filepath.Join("rules", filename)
		}
		relPaths = append(relPaths, filepath.ToSlash(rel))
	}
	return relPaths, nil
}
