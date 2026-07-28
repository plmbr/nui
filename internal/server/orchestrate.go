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
	"nui/internal/model"
	"nui/internal/skills"
	"nui/internal/store"

	"github.com/google/uuid"
)

type orchestrateRequest struct {
	Prompt     string `json:"prompt"`
	WorkingDir string `json:"workingDir,omitempty"`
}

type orchestrateResponse struct {
	Session           model.Session `json:"session"`
	Prompt            string        `json:"prompt"`
	SelectedAgentType string        `json:"selectedAgentType"`
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

	sidebarClosed := false
	setBootstrap(result.Session.ID, result.Prompt, &sidebarClosed, false)
	writeJSON(w, http.StatusCreated, orchestrateResponse{
		Session:           result.Session,
		Prompt:            result.Prompt,
		SelectedAgentType: result.Session.AgentType,
	})
}

type orchestrateRunResult struct {
	Session model.Session
	Prompt  string
}

func runOrchestrator(ctx context.Context, prompt, workingDir string) (orchestrateRunResult, error) {
	if direct, ok, err := tryDirectOrchestratorLaunch(prompt, workingDir); err != nil {
		return orchestrateRunResult{}, err
	} else if ok {
		return direct, nil
	}

	settings, err := store.LoadSettings()
	if err != nil {
		settings = store.Settings{Theme: "light"}
	}
	def := agents.OrchestratorDefinition(settings)

	ephemeralID := agent.EphemeralProjectID("orchestrate-" + uuid.NewString())
	defer extensionManager.Stop(ephemeralID)

	systemPrompt, mcpServers, err := launcherOrchestratorHarness(def, workingDir, ephemeralID)
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
		if savedAgent || !orchestratorLaunchSessionTool(ev) {
			continue
		}
		if parsed, ok := parseLaunchSessionToolResult(ev.Content); ok {
			launchResult = parsed
		}
	}
	if err := <-errCh; err != nil {
		return orchestrateRunResult{}, fmt.Errorf("orchestrator run: %w", err)
	}

	if savedAgent || launchResult.Session.ID == "" {
		s, createErr := createOrchestratorSession(workingDir, settings)
		if createErr != nil {
			return orchestrateRunResult{}, fmt.Errorf("orchestrator fallback: %w", createErr)
		}
		return orchestrateRunResult{Session: s, Prompt: prompt}, nil
	}
	if launchResult.Prompt == "" {
		launchResult.Prompt = prompt
	}

	settings, loadErr := store.LoadSettings()
	if loadErr != nil {
		settings = store.Settings{Theme: "light"}
	}
	saveSessionPreferences(launchResult.Session.AgentType, launchResult.Session.ID, settings)

	return launchResult, nil
}

func createOrchestratorSession(workingDir string, settings store.Settings) (model.Session, error) {
	s, err := createSession("", workingDir, agents.OrchestratorAgentID, nil)
	if err != nil {
		return model.Session{}, err
	}
	saveSessionPreferences(agents.OrchestratorAgentID, s.ID, settings)
	return s, nil
}

func launcherOrchestratorHarness(def model.ADLDefinition, workingDir, sessionID string) (string, []model.ADLMCPServer, error) {
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
	return prompt, deps.MCPServers, nil
}

func orchestratorSavedAgent(ev agent.Event) bool {
	if ev.Type != agent.EventToolCallResult {
		return false
	}
	return strings.Contains(strings.ToLower(ev.ToolName), "save_agent")
}

func orchestratorLaunchSessionTool(ev agent.Event) bool {
	if ev.Type != agent.EventToolCallResult {
		return false
	}
	name := strings.ToLower(ev.ToolName)
	return strings.Contains(name, "launch_session")
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
		Session: payload.Session,
		Prompt:  payload.Prompt,
	}, true
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
