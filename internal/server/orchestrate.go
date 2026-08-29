// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"nui/internal/agent"
	"nui/internal/agents"
	"nui/internal/extensions"
	"nui/internal/mcpclient"
	"nui/internal/model"
	"nui/internal/skills"
	"nui/internal/store"
	"nui/internal/uiaction"

	"github.com/google/uuid"
)

type orchestrateRequest struct {
	Prompt     string `json:"prompt"`
	WorkingDir string `json:"workingDir,omitempty"`
}

type orchestrateCandidate struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Score       int    `json:"score"`
}

type orchestrateResponse struct {
	Session           *model.Session         `json:"session,omitempty"`
	Prompt            string                 `json:"prompt,omitempty"`
	SelectedAgentType string                 `json:"selectedAgentType,omitempty"`
	Ambiguous         bool                   `json:"ambiguous,omitempty"`
	Candidates        []orchestrateCandidate `json:"candidates,omitempty"`
	UIActions         []uiaction.Action      `json:"uiActions,omitempty"`
}

func handleOrchestrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req orchestrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}

	workingDir := strings.TrimSpace(req.WorkingDir)
	if workingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workingDir = cwd
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()

	result, err := runOrchestrator(ctx, prompt, workingDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if result.Ambiguous {
		writeJSON(w, http.StatusOK, orchestrateResponse{
			Prompt:     result.Prompt,
			Ambiguous:  true,
			Candidates: result.Candidates,
		})
		return
	}

	// UI-only control (navigate/theme) — no session required.
	if result.Session.ID == "" && len(result.UIActions) > 0 {
		writeJSON(w, http.StatusOK, orchestrateResponse{
			UIActions: result.UIActions,
		})
		return
	}

	if result.Session.ID == "" {
		http.Error(w, "orchestrator did not return a session or UI actions", http.StatusBadGateway)
		return
	}

	sidebarClosed := false
	setBootstrap(result.Session.ID, result.Prompt, &sidebarClosed, false)
	sess := result.Session
	writeJSON(w, http.StatusCreated, orchestrateResponse{
		Session:           &sess,
		Prompt:            result.Prompt,
		SelectedAgentType: result.Session.AgentType,
		UIActions:         result.UIActions,
	})
}

type orchestrateRunResult struct {
	Session    model.Session
	Prompt     string
	Ambiguous  bool
	Candidates []orchestrateCandidate
	UIActions  []uiaction.Action
	// launchSeen is true when launch_session succeeded (prompt may be empty).
	launchSeen bool
}

func runOrchestrator(ctx context.Context, prompt, workingDir string) (orchestrateRunResult, error) {
	// LLM-first: do not short-circuit via heuristic tryDirectOrchestratorLaunch.
	settings, err := store.LoadSettings()
	if err != nil {
		settings = store.Settings{Theme: "light"}
	}
	def := agents.OrchestratorDefinition(settings)

	ephemeralID := agent.EphemeralProjectID("orchestrate-" + uuid.NewString())
	defer extensionManager.Stop(ephemeralID)

	systemPrompt, mcpServers, err := launcherOrchestratorHarness(def, workingDir, ephemeralID, prompt)
	if err != nil {
		return orchestrateRunResult{}, fmt.Errorf("orchestrator harness: %w", err)
	}

	adlAg := agent.NewADLAgent(def, ephemeralID, extensionManager)
	runReq := agent.RunRequest{
		NuiSessionID: ephemeralID,
		RunID:        uuid.NewString(),
		WorkingDir:   workingDir,
		Message:      prompt,
		SystemPrompt: systemPrompt,
		MCPServers:   mcpServers,
	}

	events := make(chan agent.Event, 128)
	errCh := make(chan error, 1)
	go func() {
		defer close(events)
		errCh <- adlAg.RunEphemeral(ctx, runReq, events)
	}()

	var launchResult orchestrateRunResult
	var savedAgent bool
	for ev := range events {
		if ev.Type != agent.EventToolCallResult {
			continue
		}
		if orchestratorSavedAgent(ev) {
			savedAgent = true
			continue
		}
		if actions, ok := parseControlUIToolResult(ev); ok {
			launchResult.UIActions = append(launchResult.UIActions, actions...)
			continue
		}
		if actions, ok := parseExtensionEnabledToolResult(ev); ok {
			launchResult.UIActions = append(launchResult.UIActions, actions...)
			continue
		}
		if savedAgent || !orchestratorLaunchSessionTool(ev) {
			continue
		}
		if parsed, ok := parseLaunchSessionToolResult(ev.Content); ok {
			launchResult.Session = parsed.Session
			launchResult.Prompt = parsed.Prompt
			launchResult.launchSeen = true
		}
	}
	if err := <-errCh; err != nil {
		return orchestrateRunResult{}, fmt.Errorf("orchestrator run: %w", err)
	}

	if launchResult.launchSeen {
		saveSessionPreferences(launchResult.Session)
		return launchResult, nil
	}

	// UI-only actions without opening a chat session.
	if len(launchResult.UIActions) > 0 && !savedAgent {
		return launchResult, nil
	}

	// If the model skipped tools but TF-IDF has a clear specialist match, launch it.
	// Fixes home turns that answer "I don't have that" instead of routing.
	if !savedAgent {
		if direct, ok, err := tryClearSearchLaunch(prompt, workingDir); err != nil {
			return orchestrateRunResult{}, err
		} else if ok {
			return direct, nil
		}
	}

	// Chat reply / save_agent / unanswered — open a nui session with the original prompt.
	s, createErr := createOrchestratorSession(workingDir, settings)
	if createErr != nil {
		return orchestrateRunResult{}, fmt.Errorf("orchestrator fallback: %w", createErr)
	}
	return orchestrateRunResult{
		Session:   s,
		Prompt:    prompt,
		UIActions: launchResult.UIActions,
	}, nil
}

func createOrchestratorSession(workingDir string, settings store.Settings) (model.Session, error) {
	s, err := createSession("", workingDir, agents.OrchestratorAgentID, nil)
	if err != nil {
		return model.Session{}, err
	}
	saveSessionPreferences(s)
	return s, nil
}

func launcherOrchestratorHarness(def model.ADLDefinition, workingDir, sessionID, userPrompt string) (string, []model.ADLMCPServer, error) {
	deps, err := agent.ExpandHarnessDeps(
		agent.HarnessDeps{WorkingDir: workingDir, SystemPrompt: def.SystemPrompt},
		extensions.Default,
		sessionID,
		def,
		nil,
	)
	if err != nil {
		return "", nil, err
	}
	var prompt string
	if def.Harness.Type == "api" {
		prompt = agent.APISystemPromptFromDeps(deps)
	} else {
		prompt = deps.SystemPrompt
		if appendix := skills.PromptAppendix(skills.Context{WorkingDir: workingDir}, deps.Skills); appendix != "" {
			prompt = strings.TrimSpace(prompt + "\n\n" + appendix)
		}
	}
	prompt = strings.TrimSpace(prompt + "\n\n" + agents.LauncherPromptAppendix)
	if candidates := formatLauncherSearchCandidates(userPrompt); candidates != "" {
		prompt = strings.TrimSpace(prompt + "\n\n" + candidates)
	}
	return prompt, deps.MCPServers, nil
}

// formatLauncherSearchCandidates precomputes TF-IDF top hits for the home prompt
// so the model can launch without a discovery tool round-trip.
func formatLauncherSearchCandidates(userPrompt string) string {
	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt == "" {
		return ""
	}
	hits := searchOrchestratorAgents(userPrompt, 5)
	if len(hits) == 0 {
		return ""
	}
	// Skip if nothing scored — avoid dumping zero-score noise.
	topScore, _ := hits[0]["score"].(int)
	if topScore <= 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Candidate agents for this request (precomputed search)\n\n")
	b.WriteString("These are the top matches for the user's message. If one clearly fits, call launch_session with its id immediately. Do not say you lack access to capabilities these agents provide.\n\n")
	for i, h := range hits {
		score, _ := h["score"].(int)
		if score <= 0 {
			continue
		}
		id, _ := h["id"].(string)
		label, _ := h["label"].(string)
		desc, _ := h["description"].(string)
		fmt.Fprintf(&b, "%d. id=%s | %s | score=%d\n   %s\n", i+1, id, label, score, desc)
	}
	return strings.TrimSpace(b.String())
}

// tryClearSearchLaunch launches when TF-IDF has a single clear winner (same thresholds
// as matchOrchestratorAgent). Used after the home LLM turn if launch_session was skipped.
func tryClearSearchLaunch(prompt, workingDir string) (orchestrateRunResult, bool, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return orchestrateRunResult{}, false, nil
	}
	candidates := orchestratorLaunchableAgents(listAgentTypes())
	agentInfo, score, ok := matchOrchestratorAgent(prompt, candidates)
	if !ok || score < 80 {
		return orchestrateRunResult{}, false, nil
	}
	delegated := explicitDelegatedPrompt(prompt)
	if agentInfo.PromptMode == model.ADLPromptModeAuto {
		delegated = resolveAgentLaunchPrompt(agentInfo, delegated)
	} else if delegated == "" {
		// Whole utterance is the task for the specialist (e.g. "list my project tasks").
		delegated = prompt
	}
	s, err := createSession("", workingDir, agentInfo.ID, nil)
	if err != nil {
		return orchestrateRunResult{}, false, err
	}
	saveSessionPreferences(s)
	return orchestrateRunResult{Session: s, Prompt: delegated, launchSeen: true}, true, nil
}

func orchestratorSavedAgent(ev agent.Event) bool {
	if ev.Type != agent.EventToolCallResult {
		return false
	}
	return strings.EqualFold(mcpclient.BareToolName(ev.ToolName), "save_agent")
}

func isLaunchSessionToolName(toolName string) bool {
	return strings.EqualFold(mcpclient.BareToolName(toolName), "launch_session")
}

func isControlUIToolName(toolName string) bool {
	return strings.EqualFold(mcpclient.BareToolName(toolName), "control_ui")
}

func isSetExtensionEnabledToolName(toolName string) bool {
	return strings.EqualFold(mcpclient.BareToolName(toolName), "set_extension_enabled")
}

func orchestratorLaunchSessionTool(ev agent.Event) bool {
	if ev.Type != agent.EventToolCallResult {
		return false
	}
	return isLaunchSessionToolName(ev.ToolName)
}

func parseLaunchSessionToolResult(content string) (orchestrateRunResult, bool) {
	content = strings.TrimSpace(content)
	if content == "" || strings.HasPrefix(strings.ToLower(content), "error:") {
		return orchestrateRunResult{}, false
	}
	var payload struct {
		Session model.Session `json:"session"`
		Prompt  string        `json:"prompt"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return orchestrateRunResult{}, false
	}
	if payload.Session.ID == "" {
		return orchestrateRunResult{}, false
	}
	return orchestrateRunResult{
		Session:    payload.Session,
		Prompt:     payload.Prompt,
		launchSeen: true,
	}, true
}

func parseControlUIToolResult(ev agent.Event) ([]uiaction.Action, bool) {
	if !isControlUIToolName(ev.ToolName) {
		return nil, false
	}
	return parseUIActionsFromToolContent(ev.Content)
}

func parseExtensionEnabledToolResult(ev agent.Event) ([]uiaction.Action, bool) {
	if !isSetExtensionEnabledToolName(ev.ToolName) {
		return nil, false
	}
	return parseUIActionsFromToolContent(ev.Content)
}

func parseUIActionsFromToolContent(content string) ([]uiaction.Action, bool) {
	content = strings.TrimSpace(content)
	if content == "" || strings.HasPrefix(strings.ToLower(content), "error:") {
		return nil, false
	}
	var payload struct {
		Actions []uiaction.Action `json:"actions"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, false
	}
	if len(payload.Actions) == 0 {
		return nil, false
	}
	var out []uiaction.Action
	for _, a := range payload.Actions {
		if uiaction.Validate(a) == "" {
			out = append(out, a)
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func defaultnuiAPIURL() string {
	return agent.DefaultNuiAPIURL()
}

func resolveAgentDefinition(agentType string) (model.ADLDefinition, bool) {
	def, ok := agents.LookupDefinition(agentType)
	if !ok {
		return model.ADLDefinition{}, false
	}
	if agents.IsOrchestratorAgent(agentType) {
		settings, err := store.LoadSettings()
		if err != nil {
			settings = store.Settings{Theme: "light"}
		}
		return agents.OrchestratorDefinition(settings), true
	}
	return def, true
}
