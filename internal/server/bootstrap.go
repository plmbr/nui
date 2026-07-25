// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"nui/internal/agent"
	"nui/internal/agents"
	"nui/internal/browser"
	"nui/internal/extensions"
	"nui/internal/model"
	"nui/internal/pathutil"
	"nui/internal/store"

	"github.com/google/uuid"
)

// LaunchPath is the home launcher route in the web UI.
const LaunchPath = "/launch"

// StartOptions configures optional CLI-driven session bootstrap on server start.
type StartOptions struct {
	AgentType        string
	Prompt           string
	WorkingDir       string
	Open             bool // open the UI in the system default browser
	HideInput        bool // hide the chat input in the UI (one-off runs)
	Theme            string // "light" | "dark"; persisted to settings when set
	DefaultAgentType string // ADL agent id; persisted to settings when set
	DefaultHarness   string // harness ref for internal agents; persisted to settings when set
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

// applyStartSettings persists CLI-provided theme and default agent type to settings.json.
func applyStartSettings(opts StartOptions) error {
	theme := strings.TrimSpace(opts.Theme)
	defaultAgent := strings.TrimSpace(opts.DefaultAgentType)
	defaultHarness := strings.TrimSpace(opts.DefaultHarness)
	if theme == "" && defaultAgent == "" && defaultHarness == "" {
		return nil
	}

	settings, err := store.LoadSettings()
	if err != nil {
		settings = store.Settings{Theme: "light"}
	}

	if theme != "" {
		if theme != "light" && theme != "dark" {
			return fmt.Errorf("theme must be %q or %q", "light", "dark")
		}
		settings.Theme = theme
	}
	if defaultAgent != "" {
		def, ok := findADLDef(defaultAgent)
		if !ok {
			return fmt.Errorf("unknown agent id %q", defaultAgent)
		}
		settings.DefaultAgentType = model.ADLAgentID(def)
	}
	if defaultHarness != "" {
		if !agents.HarnessAvailable(defaultHarness) {
			return fmt.Errorf("harness %q is not available on this system", defaultHarness)
		}
		settings.DefaultHarness = defaultHarness
	}
	if settings.Theme == "" {
		settings.Theme = "light"
	}
	return store.SaveSettings(settings)
}

func handleBootstrap(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, takeBootstrap())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type launchRequest struct {
	AgentType  string
	WorkingDir string
	Prompt     string
	HideInput  bool
}

type launchResult struct {
	Session        model.Session
	ResolvedPrompt string
	HideInput      bool
}

// launchSessionFromRequest creates a session using the same rules as POST /api/launch.
func launchSessionFromRequest(req launchRequest) (launchResult, error) {
	agentType := strings.TrimSpace(req.AgentType)
	if agentType == "" {
		settings, err := store.LoadSettings()
		if err != nil {
			settings = store.Settings{Theme: "light"}
		}
		agentType = ensureDefaultAgentType(&settings)
		if agentType == "" {
			return launchResult{}, fmt.Errorf("no available agent type")
		}
	}

	def, ok := findADLDef(agentType)
	if !ok {
		return launchResult{}, fmt.Errorf("unknown agent id %q", agentType)
	}

	workingDir := strings.TrimSpace(req.WorkingDir)
	if workingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workingDir = cwd
		}
	}

	s, err := createSession("", workingDir, model.ADLAgentID(def), nil)
	if err != nil {
		return launchResult{}, err
	}

	prompt := strings.TrimSpace(req.Prompt)
	hideInput := req.HideInput
	if model.IsADLAutoPrompt(def) {
		hideInput = true
		prompt = model.ResolveADLLaunchPrompt(def, prompt)
	}

	settings, loadErr := store.LoadSettings()
	if loadErr != nil {
		settings = store.Settings{Theme: "light"}
	}
	saveSessionPreferences(model.ADLAgentID(def), s.ID, settings)

	return launchResult{
		Session:        s,
		ResolvedPrompt: prompt,
		HideInput:      hideInput,
	}, nil
}

func handleLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentType  string `json:"agentType,omitempty"`
		WorkingDir string `json:"workingDir,omitempty"`
		Prompt     string `json:"prompt,omitempty"`
		HideInput  bool   `json:"hideInput,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := launchSessionFromRequest(launchRequest{
		AgentType:  req.AgentType,
		WorkingDir: req.WorkingDir,
		Prompt:     req.Prompt,
		HideInput:  req.HideInput,
	})
	if err != nil {
		if strings.Contains(err.Error(), "no available agent type") {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		if strings.Contains(err.Error(), "unknown agent id") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("agent unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}

	sidebarClosed := false
	setBootstrap(result.Session.ID, result.ResolvedPrompt, &sidebarClosed, result.HideInput)
	writeJSON(w, http.StatusCreated, result.Session)
}

// needsCLILaunch reports whether CLI flags request a session launch on server start.
func needsCLILaunch(opts StartOptions) bool {
	return strings.TrimSpace(opts.AgentType) != "" || strings.TrimSpace(opts.Prompt) != ""
}

// needsCLIOpen reports whether CLI flags request opening the browser without a session launch.
func needsCLIOpen(opts StartOptions) bool {
	return opts.Open && !needsCLILaunch(opts)
}

// runCLILaunch creates a session after the HTTP server is listening, matching POST /api/launch.
func runCLILaunch(port int, opts StartOptions) {
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	waitForHealth(baseURL)

	result, err := launchSessionFromRequest(launchRequest{
		AgentType:  opts.AgentType,
		WorkingDir: opts.WorkingDir,
		Prompt:     opts.Prompt,
		HideInput:  opts.HideInput,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "launch: %v\n", err)
		return
	}

	sidebar := cliLaunchSidebarOpen(opts)
	setBootstrap(result.Session.ID, result.ResolvedPrompt, sidebar, result.HideInput)
	fmt.Fprintf(os.Stderr, "Created session %q (%s)\n", result.Session.Name, result.Session.ID)

	if opts.Open {
		sessionURL := fmt.Sprintf("%s/sessions/%s", baseURL, result.Session.ID)
		if err := browser.Open(sessionURL); err != nil {
			fmt.Fprintf(os.Stderr, "warn: open browser: %v\n", err)
		}
	}
}

func runCLIOpen(port int) {
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	waitForHealth(baseURL)
	if err := browser.Open(baseURL + LaunchPath); err != nil {
		fmt.Fprintf(os.Stderr, "warn: open browser: %v\n", err)
	}
}

func handleEnsureDefaultSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s, err := getDefaultSession()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func getDefaultSession() (model.Session, error) {
	settings, err := store.LoadSettings()
	if err != nil {
		settings = store.Settings{Theme: "light"}
	}

	mu.RLock()
	defer mu.RUnlock()

	if len(sessions) == 0 {
		return model.Session{}, fmt.Errorf("no sessions")
	}
	if settings.LastSessionID != "" {
		for _, s := range sessions {
			if s.ID == settings.LastSessionID {
				return s, nil
			}
		}
	}
	return sessions[0], nil
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

	for _, def := range agents.BuiltinAgentDefs() {
		id := model.ADLAgentID(def)
		if seen[id] || !harnessAvailable(def) {
			continue
		}
		candidates = append(candidates, id)
		seen[id] = true
	}

	return candidates
}

func firstAvailableAPIBuiltinID() string {
	for _, want := range agents.APIBuiltinOrder {
		for _, def := range agents.BuiltinAgentDefs() {
			if def.ID != want || def.Harness.Type != "api" {
				continue
			}
			if harnessAvailable(def) {
				return want
			}
		}
	}
	return ""
}

// ensureDefaultAgentType resolves the configured default agent, persisting the
// first available built-in when settings.json has no defaultAgentType yet.
func ensureDefaultAgentType(settings *store.Settings) string {
	if settings.DefaultAgentType != "" {
		if def, ok := findADLDef(settings.DefaultAgentType); ok {
			if harnessAvailable(def) {
				return model.ADLAgentID(def)
			}
		}
	}

	if id := firstAvailableAPIBuiltinID(); id != "" {
		settings.DefaultAgentType = id
		if err := store.SaveSettings(*settings); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save default agent type: %v\n", err)
		}
		return id
	}

	for _, def := range agents.BuiltinAgentDefs() {
		id := model.ADLAgentID(def)
		if def.Harness.Type == "api" {
			continue
		}
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
	return createSessionEx(sessionCreateOpts{
		Name:        name,
		WorkingDir:  workingDir,
		AgentType:   agentType,
		AgentConfig: agentConfig,
	})
}

type sessionCreateOpts struct {
	Name         string
	WorkingDir   string
	AgentType    string
	AgentConfig  map[string]any
	ScheduleID   string
	ScheduleName string
}

func createSessionEx(opts sessionCreateOpts) (model.Session, error) {
	name := opts.Name
	workingDir := opts.WorkingDir
	agentType := opts.AgentType
	agentConfig := opts.AgentConfig
	if agentType == "" {
		return model.Session{}, fmt.Errorf("agentType is required")
	}
	if agents.IsInternalAgent(agentType) {
		return model.Session{}, fmt.Errorf("agent %q is internal and cannot be used for user sessions", agentType)
	}
	def, ok := resolveAgentDefinition(agentType)
	if !ok {
		return model.Session{}, fmt.Errorf("unknown agent type: %s", agentType)
	}
	if strings.TrimSpace(name) == "" {
		name = PendingSessionTitle
	}

	sessionID := uuid.NewString()
	resolvedWorkingDir, err := resolveSessionWorkingDir(sessionID, def, workingDir)
	if err != nil {
		return model.Session{}, fmt.Errorf("prepare working directory: %w", err)
	}

	s := model.Session{
		ID:           sessionID,
		Name:         name,
		WorkingDir:   resolvedWorkingDir,
		AgentType:    model.ADLAgentID(def),
		AgentConfig:  agentConfig,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		ScheduleID:   strings.TrimSpace(opts.ScheduleID),
		ScheduleName: strings.TrimSpace(opts.ScheduleName),
	}
	if err := validateSessionConnector(s); err != nil {
		if !model.IsADLWorkingDirInput(def) {
			_ = store.RemoveSessionWorkspace(sessionID)
		}
		return model.Session{}, err
	}

	if err := agent.PrepareSessionHarnessConfig(sessionID, def, extensions.Default, agentConfig); err != nil {
		fmt.Fprintf(os.Stderr, "warning: provision session harness config: %v\n", err)
	}

	mu.Lock()
	sessions = append(sessions, s)
	snapshot := snapshotData()
	mu.Unlock()
	if err := store.SaveData(snapshot); err != nil {
		fmt.Fprintf(os.Stderr, "warn: save data after create: %v\n", err)
	}
	notifySessionsChanged()
	return s, nil
}

func resolveSessionWorkingDir(sessionID string, def model.ADLDefinition, requested string) (string, error) {
	if model.IsADLWorkingDirInput(def) {
		return pathutil.ExpandHome(requested)
	}
	return store.EnsureSessionWorkspace(sessionID)
}

// resolveUserPromptAgentType picks an agent for URL-driven session creation.
// Auto-prompt agents are rejected because URL sessions always wait for user input.
func resolveUserPromptAgentType(preferred string) (model.ADLDefinition, error) {
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		def, ok := findADLDef(preferred)
		if !ok {
			return model.ADLDefinition{}, fmt.Errorf("unknown agent id %q", preferred)
		}
		if model.IsADLAutoPrompt(def) {
			return model.ADLDefinition{}, fmt.Errorf("agent %q uses auto prompt mode and cannot be used for URL session creation", preferred)
		}
		return def, nil
	}

	for _, id := range defaultAgentTypeCandidates() {
		def, ok := findADLDef(id)
		if !ok || model.IsADLAutoPrompt(def) {
			continue
		}
		return def, nil
	}
	return model.ADLDefinition{}, fmt.Errorf("no user-prompt agent available")
}

func handleNewSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	def, err := resolveUserPromptAgentType(r.URL.Query().Get("agent"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workingDir := strings.TrimSpace(r.URL.Query().Get("cwd"))
	if workingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workingDir = cwd
		}
	}

	s, err := createSession("", workingDir, model.ADLAgentID(def), nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("agent unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}

	settings, loadErr := store.LoadSettings()
	if loadErr != nil {
		settings = store.Settings{Theme: "light"}
	}
	saveSessionPreferences(model.ADLAgentID(def), s.ID, settings)
	writeJSON(w, http.StatusCreated, s)
}
