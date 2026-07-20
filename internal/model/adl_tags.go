// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"fmt"
	"strings"
)

// NormalizeADLTags trims, deduplicates, and drops empty agent tags.
func NormalizeADLTags(def *ADLDefinition) {
	if def == nil || len(def.Tags) == 0 {
		return
	}
	seen := make(map[string]bool, len(def.Tags))
	out := make([]string, 0, len(def.Tags))
	for _, tag := range def.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	def.Tags = out
}

// ValidateADLTags checks top-level agent tags for structural errors.
func ValidateADLTags(tags []string) error {
	seen := make(map[string]bool, len(tags))
	for i, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return fmt.Errorf("tags[%d]: tag must not be empty", i)
		}
		if seen[tag] {
			return fmt.Errorf("tags[%d]: duplicate tag %q", i, tag)
		}
		seen[tag] = true
	}
	return nil
}
