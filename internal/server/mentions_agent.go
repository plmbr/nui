// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"nui/internal/extensions"
	"nui/internal/model"
)

func allowedMentionRootsForAgent(agentType string) map[string]bool {
	def, ok := findADLDef(agentType)
	if !ok || extensions.Default == nil {
		return nil
	}
	roots, err := extensions.Default.ExpandMentionProviders(def.AIAssets.MentionProviders)
	if err != nil {
		return nil
	}
	if len(roots) == 0 {
		return nil
	}
	out := make(map[string]bool, len(roots))
	for _, root := range roots {
		out[root] = true
	}
	return out
}

func mentionAllowedRoots(session model.Session) map[string]bool {
	return allowedMentionRootsForAgent(session.AgentType)
}
