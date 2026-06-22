// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"loop/internal/agent"
	"loop/internal/model"
	"loop/internal/store"

	"github.com/google/uuid"
)

// StartOptions configures optional CLI-driven session bootstrap on server start.
type StartOptions struct {
	AgentType  string
	Prompt     string
	WorkingDir string
	Open       bool // open the UI in the system default browser
	HideInput  bool // hide the chat input in the UI (one-off runs)
}

type bootstrapState struct {
	SessionID     string `json:"sessionId,omitempty"`
	InitialPrompt string `json:"initialPrompt,omitempty"`
	SidebarOpen   *bool  `json:"sidebarOpen,omitempty"`
	HideInput     bool   `json:"hideInput,omitempty"`
}

var (
	bootstrapMu sync.Mutex
	bootstrap   bootstrapState
)

func setBootstrap(sessionID, prompt string, sidebarOpen *bool, hideInput bool) {
	bootstrapMu.Lock()
	bootstrap.SessionID = sessionID
	bootstrap.InitialPrompt = prompt
	bootstrap.SidebarOpen = sidebarOpen
	bootstrap.HideInput = hideInput
	bootstrapMu.Unlock()
}

func cliLaunchSidebarOpen(opts StartOptions) *bool {
	if strings.TrimSpace(opts.AgentType) != "" || strings.TrimSpace(opts.Prompt) != "" {
		closed := false
		return &closed
	}
	return nil
}

func takeBootstrap() bootstrapState {
	bootstrapMu.Lock()
	defer bootstrapMu.Unlock()
	out := bootstrap
	bootstrap = bootstrapState{}
	return out
}

func handleBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, takeBootstrap())
}

// bootstrapFromCLI creates a session from CLI flags and exposes it to the UI via bootstrap.
func bootstrapFromCLI(opts StartOptions) error {
	agentType := strings.TrimSpace(opts.AgentType)
	if agentType != "" {
		def, ok := findADLDef(agentType)
		if !ok {
			return fmt.Errorf("unknown agent id %q", agentType)
		}

		workingDir := strings.TrimSpace(opts.WorkingDir)
		if workingDir == "" {
			if cwd, err := os.Getwd(); err == nil {
				workingDir = cwd
			}
		}

		s, err := createSession(model.ADLAgentLabel(def), workingDir, model.ADLAgentID(def), nil)
		if err != nil {
			return err
		}

		prompt := strings.TrimSpace(opts.Prompt)
		hideInput := opts.HideInput
		if model.IsADLAutoPrompt(def) {
			hideInput = true
			prompt = model.ResolveADLLaunchPrompt(def, prompt)
		}

		setBootstrap(s.ID, prompt, cliLaunchSidebarOpen(opts), hideInput)

		settings, err := store.LoadSettings()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: load settings for bootstrap: %v\n", err)
			settings = store.Settings{Theme: "light"}
		}
		saveSessionPreferences(model.ADLAgentID(def), s.ID, settings)

		fmt.Fprintf(os.Stderr, "Created session %q (%s) with agent %s\n", s.Name, s.ID, model.ADLAgentID(def))
		return nil
	}

	s, err := createDefaultSession()
	if err != nil {
		return fmt.Errorf("create default session: %w", err)
	}
	setBootstrap(s.ID, "", nil, false)
	return nil
}

// ensureDefaultSession creates a session when none exist, using the preferred default agent.
func ensureDefaultSession() {
	mu.RLock()
	count := len(sessions)
	mu.RUnlock()
	if count > 0 {
		return
	}
	if _, err := createDefaultSession(); err != nil {
		fmt.Fprintf(os.Stderr, "warn: %v\n", err)
	}
}

func handleEnsureDefaultSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s, err := getOrCreateDefaultSession()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func getOrCreateDefaultSession() (model.Session, error) {
	settings, err := store.LoadSettings()
	if err != nil {
		settings = store.Settings{Theme: "light"}
	}

	mu.RLock()
	if len(sessions) == 0 {
		mu.RUnlock()
		return createDefaultSession()
	}
	if settings.LastSessionID != "" {
		for _, s := range sessions {
			if s.ID == settings.LastSessionID {
				mu.RUnlock()
				return s, nil
			}
		}
	}
	s := sessions[0]
	mu.RUnlock()
	saveSessionPreferences(s.AgentType, s.ID, settings)
	return s, nil
}

func createDefaultSession() (model.Session, error) {
	workingDir := ""
	if cwd, err := os.Getwd(); err == nil {
		workingDir = cwd
	}

	for _, agentType := range defaultAgentTypeCandidates() {
		s, err := createSession(agentType, workingDir, agentType, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: could not create default session with %s: %v\n", agentType, err)
			continue
		}

		settings, loadErr := store.LoadSettings()
		if loadErr != nil {
			settings = store.Settings{Theme: "light"}
		}
		saveSessionPreferences(agentType, s.ID, settings)

		fmt.Fprintf(os.Stderr, "Created default session %q (%s) with agent %s\n", s.Name, s.ID, agentType)
		return s, nil
	}

	return model.Session{}, fmt.Errorf("no agent type available for default session")
}

func defaultAgentTypeCandidates() []string {
	settings, _ := store.LoadSettings()
	primary := ensureDefaultAgentType(&settings)

	var candidates []string
	seen := map[string]bool{}

	if primary != "" {
		candidates = append(candidates, primary)
		seen[primary] = true
	}

	for _, def := range builtinAgentDefs {
		id := model.ADLAgentID(def)
		if seen[id] || !agent.CLIAvailable(def.Harness.Type) {
			continue
		}
		candidates = append(candidates, id)
		seen[id] = true
	}

	if len(candidates) == 0 && len(builtinAgentDefs) > 0 {
		candidates = append(candidates, model.ADLAgentID(builtinAgentDefs[0]))
	}
	return candidates
}

// ensureDefaultAgentType resolves the configured default agent, persisting the
// first available built-in when settings.json has no defaultAgentType yet.
func ensureDefaultAgentType(settings *store.Settings) string {
	if settings.DefaultAgentType != "" {
		if def, ok := findADLDef(settings.DefaultAgentType); ok {
			return model.ADLAgentID(def)
		}
	}

	for _, def := range builtinAgentDefs {
		id := model.ADLAgentID(def)
		if !agent.CLIAvailable(def.Harness.Type) {
			continue
		}
		settings.DefaultAgentType = id
		if err := store.SaveSettings(*settings); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save default agent type: %v\n", err)
		}
		return id
	}
	return ""
}

func saveSessionPreferences(agentType, sessionID string, settings store.Settings) {
	settings.LastAgentType = agentType
	settings.LastSessionID = sessionID
	if err := store.SaveSettings(settings); err != nil {
		fmt.Fprintf(os.Stderr, "warn: save settings after session create: %v\n", err)
	}
}

func createSession(name, workingDir, agentType string, agentConfig map[string]any) (model.Session, error) {
	if name == "" || agentType == "" {
		return model.Session{}, fmt.Errorf("name and agentType are required")
	}
	def, ok := findADLDef(agentType)
	if !ok {
		return model.Session{}, fmt.Errorf("unknown agent type: %s", agentType)
	}

	s := model.Session{
		ID:          uuid.NewString(),
		Name:        name,
		WorkingDir:  workingDir,
		AgentType:   model.ADLAgentID(def),
		AgentConfig: agentConfig,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := validateSessionConnector(s); err != nil {
		return model.Session{}, err
	}

	mu.Lock()
	sessions = append(sessions, s)
	snapshot := snapshotData()
	mu.Unlock()
	if err := store.SaveData(snapshot); err != nil {
		fmt.Fprintf(os.Stderr, "warn: save data after create: %v\n", err)
	}
	return s, nil
}
