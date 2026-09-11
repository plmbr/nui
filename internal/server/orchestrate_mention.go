// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"strings"
	"unicode"

	"nui/internal/model"
)

// parseOrchestratorAgentMention extracts a leading @agent-id mention from a launcher prompt.
// Returns the agent id (display suffix stripped), the delegated task text, and whether a mention was present.
// Prefer matchLeadingAgentMention when candidate agents are available so multi-word labels match.
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

func agentMentionKeys(agent AgentTypeInfo) []string {
	keys := make([]string, 0, 2)
	if id := strings.TrimSpace(agent.ID); id != "" {
		keys = append(keys, id)
	}
	if label := strings.TrimSpace(agent.Label); label != "" {
		dup := false
		for _, key := range keys {
			if strings.EqualFold(key, label) {
				dup = true
				break
			}
		}
		if !dup {
			keys = append(keys, label)
		}
	}
	return keys
}

// leadingMentionKeyLen reports the byte length of key when it is a case-insensitive
// prefix of rest and ends on a string/space boundary.
func leadingMentionKeyLen(rest, key string) (int, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return 0, false
	}
	restRunes := []rune(rest)
	keyRunes := []rune(key)
	if len(restRunes) < len(keyRunes) {
		return 0, false
	}
	prefix := string(restRunes[:len(keyRunes)])
	if !strings.EqualFold(prefix, key) {
		return 0, false
	}
	if len(restRunes) > len(keyRunes) && !unicode.IsSpace(restRunes[len(keyRunes)]) {
		return 0, false
	}
	return len(prefix), true
}

// matchLeadingAgentMention resolves @agent mentions against ids and labels, preferring the
// longest match so multi-word labels like "Claude Code" work when typed with spaces.
func matchLeadingAgentMention(prompt string, candidates []AgentTypeInfo) (AgentTypeInfo, string, bool) {
	prompt = strings.TrimSpace(prompt)
	if !strings.HasPrefix(prompt, "@") {
		return AgentTypeInfo{}, "", false
	}
	rest := prompt[1:]
	if rest == "" {
		return AgentTypeInfo{}, "", false
	}

	if labelStart := strings.Index(rest, ":["); labelStart >= 0 {
		closeRel := strings.Index(rest[labelStart:], "]")
		if closeRel < 0 {
			return AgentTypeInfo{}, "", false
		}
		fullEnd := labelStart + closeRel + 1
		mention := orchestratorMentionAgentID(rest[:fullEnd])
		agent, found := findAgentByMentionID(mention, candidates)
		if !found {
			return AgentTypeInfo{}, "", false
		}
		return agent, strings.TrimSpace(rest[fullEnd:]), true
	}

	bestLen := -1
	var best AgentTypeInfo
	for _, candidate := range candidates {
		for _, key := range agentMentionKeys(candidate) {
			n, ok := leadingMentionKeyLen(rest, key)
			if !ok || n < bestLen {
				continue
			}
			if n == bestLen && best.ID != "" {
				// Prefer exact id ties already set; keep first longest match stable.
				continue
			}
			bestLen = n
			best = candidate
		}
	}
	if bestLen < 0 {
		return AgentTypeInfo{}, "", false
	}
	return best, strings.TrimSpace(rest[bestLen:]), true
}

// tryMentionAgentLaunch creates a specialist session when the launcher prompt starts with @agent-id.
// No orchestrator LLM run is performed for a valid mention.
func tryMentionAgentLaunch(prompt, workingDir string) (orchestrateRunResult, bool, error) {
	candidates := orchestratorMentionableAgents(listAgentTypes())
	agent, delegated, ok := matchLeadingAgentMention(prompt, candidates)
	if !ok {
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
