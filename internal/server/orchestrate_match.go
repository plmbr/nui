// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"nui/internal/agentindex"
	"nui/internal/agents"
	"nui/internal/extensions"
	"nui/internal/model"
	"nui/internal/store"
)

var (
	orchestrateSendToRe = regexp.MustCompile(`(?i)^send(?:\s+a)?\s+(.+?)\s+to\s+`)
	orchestrateTellRe   = regexp.MustCompile(`(?i)^(?:tell|ask)\s+.+?\s+to\s+(.+)$`)
	possessiveAgentRe   = regexp.MustCompile(`(?i)(\w+)'s\s+agent`)
	toAgentRe           = regexp.MustCompile(`(?i)\bto\s+(\w+)(?:'s)?\s+agent\b`)
	agentNamedRe        = regexp.MustCompile(`(?i)\bagent\s+([\w-]+)\b`)
)

func listAgentTypes() []AgentTypeInfo {
	var all []AgentTypeInfo
	settings, _ := store.LoadSettings()
	for _, def := range agents.BuiltinAgentDefs() {
		if agents.IsOrchestratorAgent(def.ID) {
			def = agents.OrchestratorDefinition(settings)
		}
		all = append(all, agentTypeInfoFromDef(def, true))
	}

	userDefs, err := store.LoadADLDefinitions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: load ADL definitions: %v\n", err)
	}
	for _, def := range userDefs {
		all = append(all, agentTypeInfoFromDef(def, false))
	}

	if extensions.Default != nil {
		for _, def := range extensions.Default.AllAgents() {
			info := agentTypeInfoFromDef(def, false)
			info.Source = "extension"
			enrichExtensionAgentInfo(&info)
			all = append(all, info)
		}
		for _, def := range extensions.Default.HarnessOnlyAgentTypes() {
			info := agentTypeInfoFromDef(def, false)
			info.Source = "extension"
			info.Harness = "extension"
			info.Available = true
			all = append(all, info)
		}
	}
	return all
}

func orchestratorListableAgents(all []AgentTypeInfo) []AgentTypeInfo {
	var out []AgentTypeInfo
	for _, info := range all {
		if !info.Available {
			continue
		}
		if !agents.IsOrchestratorRoutingTarget(info.ID) {
			continue
		}
		out = append(out, info)
	}
	return out
}

func orchestratorLaunchableAgents(all []AgentTypeInfo) []AgentTypeInfo {
	return orchestratorListableAgents(all)
}

func resolveAgentLaunchPrompt(agent AgentTypeInfo, override string) string {
	if p := strings.TrimSpace(override); p != "" {
		return p
	}
	if p := strings.TrimSpace(agent.DefaultPrompt); p != "" {
		return p
	}
	return model.ADLDefaultAutoPrompt
}

func tryDirectOrchestratorLaunch(prompt, workingDir string) (orchestrateRunResult, bool, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return orchestrateRunResult{}, false, nil
	}
	candidates := orchestratorLaunchableAgents(listAgentTypes())
	agent, score, ok := matchOrchestratorAgent(prompt, candidates)
	if !ok || score < 80 {
		return orchestrateRunResult{}, false, nil
	}
	delegated := explicitDelegatedPrompt(prompt)
	if agent.PromptMode == model.ADLPromptModeAuto {
		delegated = resolveAgentLaunchPrompt(agent, delegated)
	} else {
		delegated = extractDelegatedPrompt(prompt, agent)
	}
	s, err := createSession("", workingDir, agent.ID, nil)
	if err != nil {
		return orchestrateRunResult{}, false, err
	}
	settings, loadErr := store.LoadSettings()
	if loadErr != nil {
		settings = store.Settings{Theme: "light"}
	}
	saveSessionPreferences(s.AgentType, s.ID, settings)
	return orchestrateRunResult{Session: s, Prompt: delegated}, true, nil
}

func matchOrchestratorAgent(prompt string, candidates []AgentTypeInfo) (AgentTypeInfo, int, bool) {
	if len(candidates) == 0 {
		return AgentTypeInfo{}, 0, false
	}
	refs := extractAgentReferences(prompt)
	semantic := tfidfAgentScores(prompt, candidates)
	bestIdx := -1
	bestScore := 0
	secondScore := 0
	for i, candidate := range candidates {
		score := agentMatchScore(prompt, refs, candidate)
		if cos, ok := semantic[candidate.ID]; ok {
			score += agentindex.SemanticBonus(cos)
		}
		if score > bestScore {
			secondScore = bestScore
			bestScore = score
			bestIdx = i
			continue
		}
		if score > secondScore {
			secondScore = score
		}
	}
	if bestIdx < 0 || bestScore < 80 {
		return AgentTypeInfo{}, 0, false
	}
	if secondScore > 0 && bestScore-secondScore < 25 {
		return AgentTypeInfo{}, 0, false
	}
	return candidates[bestIdx], bestScore, true
}

func tfidfAgentScores(prompt string, candidates []AgentTypeInfo) map[string]float64 {
	docs := make([]agentindex.Doc, 0, len(candidates))
	for _, c := range candidates {
		docs = append(docs, agentindex.BuildDoc(c.ID, c.Label, c.Description, c.Tags))
	}
	return agentindex.Score(prompt, docs)
}

func agentMatchScore(prompt string, refs []string, agent AgentTypeInfo) int {
	lower := strings.ToLower(strings.TrimSpace(prompt))
	score := 0
	localID := strings.ToLower(agentLocalID(agent.ID))
	label := strings.ToLower(agent.Label)
	desc := strings.ToLower(agent.Description)
	nameTokens := agentNameTokens(localID, label)
	searchTokens := agentSearchTokens(agent)

	for _, ref := range refs {
		ref = strings.ToLower(strings.TrimSpace(ref))
		if ref == "" {
			continue
		}
		if strings.Contains(localID, ref) || strings.Contains(label, ref) {
			score += 100
		}
		for _, tok := range nameTokens {
			if strings.Contains(tok, ref) || strings.Contains(ref, tok) {
				score += 90
			}
		}
	}

	for _, tok := range nameTokens {
		if len(tok) < 4 {
			continue
		}
		if tokenMatchesPrompt(lower, tok) {
			score += 50
			// Reward name tokens that also appear in the description only when
			// they are grounded in the prompt (avoids free boosts for unused
			// suffixes like "short").
			if strings.Contains(desc, tok) {
				score += 20
			}
		}
	}

	score += distinctTokenMatchBonus(lower, nameTokens)
	score += localIDPartMatchScore(lower, localID)
	score -= unmatchedNameTokenPenalty(lower, nameTokens)
	score += descriptionWordMatchScore(lower, desc)

	for _, word := range promptWords(lower) {
		for _, tok := range searchTokens {
			if word == tok {
				score += 45
				continue
			}
			if stemMatch(word, tok) {
				score += 55
			}
		}
	}

	for _, tag := range agent.Tags {
		for _, word := range agentNameTokens("", tag) {
			if len(word) >= 4 && strings.Contains(lower, word) {
				score += 35
			}
		}
	}

	if agent.PromptMode == model.ADLPromptModeAuto {
		score += defaultPromptMatchScore(lower, agent.DefaultPrompt)
	}

	if localIDSubstringBonus(lower, localID) {
		score += 120
	}
	return score
}

func localIDSubstringBonus(prompt, localID string) bool {
	localID = strings.ToLower(strings.TrimSpace(localID))
	if localID == "" || looksLikeUUID(localID) {
		return false
	}
	if !strings.Contains(prompt, localID) {
		return false
	}
	// Short ids like "acme" match too many task prompts; require specificity.
	if len(localID) < 10 && !strings.ContainsAny(localID, "-_") {
		return false
	}
	return true
}

func descriptionWordMatchScore(prompt, desc string) int {
	descWords := promptWords(desc)
	if len(descWords) == 0 {
		return 0
	}
	inDesc := map[string]bool{}
	for _, w := range descWords {
		inDesc[w] = true
	}
	score := 0
	for _, word := range promptWords(prompt) {
		if len(word) < 4 {
			continue
		}
		if inDesc[word] {
			score += 40
		}
	}
	return score
}

// unmatchedNameTokenPenalty prefers agents whose name is covered by the prompt
// over near-duplicate variants with extra unmatched tokens (e.g. "…-short").
func unmatchedNameTokenPenalty(prompt string, nameTokens []string) int {
	penalty := 0
	for _, tok := range nameTokens {
		if len(tok) < 4 {
			continue
		}
		if genericAgentNameToken(tok) {
			continue
		}
		if tokenMatchesPrompt(prompt, tok) {
			continue
		}
		penalty += 40
	}
	return penalty
}

func genericAgentNameToken(tok string) bool {
	switch tok {
	case "agent", "runtime", "coding", "assistant", "helper", "local",
		"extension", "experimental", "internal", "builtin", "test":
		return true
	default:
		return false
	}
}

func tokenMatchesPrompt(prompt, tok string) bool {
	tok = strings.ToLower(strings.TrimSpace(tok))
	if len(tok) < 4 {
		return false
	}
	if strings.Contains(prompt, tok) {
		return true
	}
	for _, word := range promptWords(prompt) {
		if stemMatch(word, tok) {
			return true
		}
	}
	return false
}

func distinctTokenMatchBonus(prompt string, tokens []string) int {
	matched := 0
	for _, tok := range tokens {
		if tokenMatchesPrompt(prompt, tok) {
			matched++
		}
	}
	if matched < 2 {
		return 0
	}
	return (matched - 1) * 45
}

func localIDPartMatchScore(prompt, localID string) int {
	parts := agentNameTokens(localID, "")
	if len(parts) == 0 {
		return 0
	}
	matched := 0
	for _, part := range parts {
		if tokenMatchesPrompt(prompt, part) {
			matched++
		}
	}
	if matched < 2 {
		return 0
	}
	return matched * 25
}

func agentSearchTokens(agent AgentTypeInfo) []string {
	localID := strings.ToLower(agentLocalID(agent.ID))
	tokens := agentNameTokens(localID, agent.Label)
	seen := make(map[string]bool, len(tokens))
	for _, tok := range tokens {
		seen[tok] = true
	}
	add := func(raw string) {
		for _, part := range agentNameTokens("", raw) {
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			tokens = append(tokens, part)
		}
	}
	add(agent.Description)
	for _, tag := range agent.Tags {
		add(tag)
	}
	if agent.PromptMode == model.ADLPromptModeAuto {
		add(agent.DefaultPrompt)
	}
	return tokens
}

func promptWords(prompt string) []string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.FieldsFunc(prompt, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

func stemMatch(a, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	shorter, longer := a, b
	if len(shorter) > len(longer) {
		shorter, longer = longer, shorter
	}
	if len(shorter) < 4 {
		return false
	}
	return strings.HasPrefix(longer, shorter)
}

func defaultPromptMatchScore(prompt, defaultPrompt string) int {
	defaultPrompt = strings.ToLower(strings.TrimSpace(defaultPrompt))
	if defaultPrompt == "" {
		return 0
	}
	score := 0
	for _, word := range promptWords(prompt) {
		if len(word) < 4 {
			continue
		}
		if strings.Contains(defaultPrompt, word) {
			score += 30
		}
	}
	return score
}

func extractAgentReferences(prompt string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(raw string) {
		raw = strings.ToLower(strings.TrimSpace(raw))
		if raw == "" || seen[raw] {
			return
		}
		seen[raw] = true
		out = append(out, raw)
	}
	for _, m := range possessiveAgentRe.FindAllStringSubmatch(prompt, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range toAgentRe.FindAllStringSubmatch(prompt, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range agentNamedRe.FindAllStringSubmatch(prompt, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	return out
}

func agentLocalID(id string) string {
	id = strings.TrimSpace(id)
	if idx := strings.LastIndex(id, "/"); idx >= 0 {
		return id[idx+1:]
	}
	return id
}

func agentNameTokens(localID, label string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(raw string) {
		raw = strings.ToLower(strings.TrimSpace(raw))
		if raw == "" || seen[raw] || looksLikeUUID(raw) {
			return
		}
		seen[raw] = true
		out = append(out, raw)
	}
	for _, part := range strings.FieldsFunc(localID, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		add(part)
	}
	for _, part := range strings.FieldsFunc(label, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		add(part)
	}
	return out
}

func looksLikeUUID(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
				return false
			}
		}
	}
	return true
}

func explicitDelegatedPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if m := orchestrateSendToRe.FindStringSubmatch(prompt); len(m) > 1 {
		if delegated := strings.TrimSpace(m[1]); delegated != "" {
			return delegated
		}
	}
	if m := orchestrateTellRe.FindStringSubmatch(prompt); len(m) > 1 {
		if delegated := strings.TrimSpace(m[1]); delegated != "" {
			return delegated
		}
	}
	return ""
}

func extractDelegatedPrompt(prompt string, agent AgentTypeInfo) string {
	if delegated := explicitDelegatedPrompt(prompt); delegated != "" {
		return delegated
	}
	_ = agent
	return strings.TrimSpace(prompt)
}
