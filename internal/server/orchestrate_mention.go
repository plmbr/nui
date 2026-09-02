// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"strings"
	"unicode"

	"nui/internal/model"
)

// parseOrchestratorAgentMention extracts a leading @agent-id mention from a launcher prompt.
// Returns the agent id (display suffix stripped), the delegated task text, and whether a mention was present.
func parseOrchestratorAgentMention(prompt string) (mention string, delegated string, ok bool) {
	prompt = strings.TrimSpace(prompt)
	if !strings.HasPrefix(prompt, "@") {
		return "", "", false
	}
	rest := prompt[1:]
	if rest == "" {
		return "", "", false
	}

	if labelStart := strings.Index(rest, ":["); labelStart >= 0 {
		closeRel := strings.Index(rest[labelStart:], "]")
		if closeRel < 0 {
			return "", "", false
		}
		fullEnd := labelStart + closeRel + 1
		mention = orchestratorMentionAgentID(rest[:fullEnd])
		delegated = strings.TrimSpace(rest[fullEnd:])
		if mention == "" {
			return "", "", false
		}
		return mention, delegated, true
	}

	end := strings.IndexFunc(rest, unicode.IsSpace)
	if end < 0 {
		return orchestratorMentionAgentID(rest), "", true
	}
	mention = orchestratorMentionAgentID(rest[:end])
	delegated = strings.TrimSpace(rest[end:])
	if mention == "" {
		return "", "", false
	}
	return mention, delegated, true
}

func orchestratorMentionAgentID(token string) string {
	token = strings.TrimSpace(token)
	if idx := strings.Index(token, ":["); idx >= 0 && strings.HasSuffix(token, "]") {
		return token[:idx]
	}
	return token
}

func findAgentByMentionID(mention string, candidates []AgentTypeInfo) (AgentTypeInfo, bool) {
	mention = orchestratorMentionAgentID(strings.TrimSpace(mention))
	if mention == "" {
		return AgentTypeInfo{}, false
	}
	for _, candidate := range candidates {
		if candidate.ID == mention {
			return candidate, true
		}
	}
	return AgentTypeInfo{}, false
}

// tryMentionAgentLaunch creates a specialist session when the launcher prompt starts with @agent-id.
// No orchestrator LLM run is performed for a valid mention.
func tryMentionAgentLaunch(prompt, workingDir string) (orchestrateRunResult, bool, error) {
	mention, delegated, ok := parseOrchestratorAgentMention(prompt)
	if !ok {
		return orchestrateRunResult{}, false, nil
	}
	candidates := orchestratorMentionableAgents(listAgentTypes())
	agent, found := findAgentByMentionID(mention, candidates)
	if !found {
		return orchestrateRunResult{}, false, nil
	}
	s, err := createSession("", workingDir, agent.ID, nil)
	if err != nil {
		return orchestrateRunResult{}, false, err
	}
	saveSessionPreferences(s)
	launchPrompt := delegated
	if agent.PromptMode == model.ADLPromptModeAuto {
		// Auto-mode agents use ADL defaultPrompt; user text after @mention is ignored.
		launchPrompt = resolveAgentLaunchPrompt(agent, "")
	}
	return orchestrateRunResult{
		Session:    s,
		Prompt:     launchPrompt,
		launchSeen: true,
	}, true, nil
}
