// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"loop/internal/agent"
	"loop/internal/model"
	"loop/internal/store"

	"github.com/google/uuid"
)

// AgentTypeInfo is the API shape returned by GET /api/agent-types.
// ID equals the ADL definition name and is stored in Session.AgentType.
type AgentTypeInfo struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Harness     string `json:"harness"`           // claude-code | pi | codex | opencode | docker | remote
	Sandbox     string `json:"sandbox,omitempty"` // none | bubblewrap | docker
	IsBuiltin   bool   `json:"isBuiltin"`
	Available   bool   `json:"available"` // false when the required CLI is not installed
}

// builtinAgentDefs are the compiled-in ADL definitions shipped with Loop.
// They are expressed in the same ADL format as user-defined agents in ~/.loop/agents/*.yaml.
// The three subprocess-based built-ins (claude-code, pi, codex) and two connector types (docker, remote)
// correspond directly to the five step harness types. Sandbox variants live in user-defined ADL.
var builtinAgentDefs = []model.ADLDefinition{
	{
		Name:        "Claude Code",
		Description: "Claude Code running as a local subprocess",
		Harness:     model.ADLHarness{Type: "claude-code", Sandbox: "none"},
	},
	{
		Name:        "pi",
		Description: "Pi running as a local subprocess",
		Harness:     model.ADLHarness{Type: "pi", Sandbox: "none"},
	},
	{
		Name:        "codex",
		Description: "Codex running as a local subprocess",
		Harness:     model.ADLHarness{Type: "codex", Sandbox: "none"},
	},
	{
		Name:        "opencode",
		Description: "opencode running as a local subprocess",
		Harness:     model.ADLHarness{Type: "opencode", Sandbox: "none"},
	},
}

// legacyAgentTypeNames maps old Session.AgentType strings to the new ADL definition name.
var legacyAgentTypeNames = map[string]string{
	"claude-code":     "Claude Code",
	"pi":              "pi",
	"docker-claude":   "Claude Code",
	"docker-pi":       "pi",
	"docker-opencode": "opencode",
}

type SandboxCapabilities struct {
	Bwrap agent.BwrapStatus `json:"bwrap"`
}

type Capabilities struct {
	Sandbox SandboxCapabilities `json:"sandbox"`
}

var (
	mu              sync.RWMutex
	sessions        []model.Session
	sessionMessages = map[string][]model.ChatMessage{}
	agentSessions   = map[string]string{} // session ID → agent session ID
)

func initStore() error {
	data, err := store.LoadData()
	if err != nil {
		return err
	}
	mu.Lock()
	sessions = data.Sessions
	agentSessions = data.AgentSessions
	sessionMessages = data.SessionMessages
	if sessionMessages == nil {
		sessionMessages = map[string][]model.ChatMessage{}
	}
	mu.Unlock()
	if err := store.ProvisionDefaultAgents(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: provisioning default agents: %v\n", err)
	}
	return nil
}

// snapshotData must be called with mu held.
func snapshotData() store.Data {
	ss := make([]model.Session, len(sessions))
	copy(ss, sessions)
	as := make(map[string]string, len(agentSessions))
	for k, v := range agentSessions {
		as[k] = v
	}
	sm := make(map[string][]model.ChatMessage, len(sessionMessages))
	for k, v := range sessionMessages {
		copied := make([]model.ChatMessage, len(v))
		copy(copied, v)
		sm[k] = copied
	}
	return store.Data{Sessions: ss, AgentSessions: as, SessionMessages: sm}
}

func registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/sessions", handleSessions)
	mux.HandleFunc("/api/sessions/", handleSession)
	mux.HandleFunc("/api/agent-types", handleAgentTypes)
	mux.HandleFunc("/api/directories", handleDirectories)
	mux.HandleFunc("/api/settings", handleSettings)
	mux.HandleFunc("/api/capabilities", handleCapabilities)
}

var errDirectoryOutsideHome = errors.New("directory is outside the home directory")

func handleDirectories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		http.Error(w, "unable to determine home directory", http.StatusInternalServerError)
		return
	}
	handleDirectoriesForHome(w, r, home)
}

func handleDirectoriesForHome(w http.ResponseWriter, r *http.Request, home string) {
	directories, err := suggestDirectories(home, r.URL.Query().Get("path"))
	if errors.Is(err, errDirectoryOutsideHome) {
		http.Error(w, "path must be inside the home directory", http.StatusBadRequest)
		return
	}
	if err != nil {
		// Missing and unreadable directories are normal while a path is being typed.
		directories = []string{}
	}
	writeJSON(w, http.StatusOK, struct {
		Directories []string `json:"directories"`
	}{Directories: directories})
}

func suggestDirectories(home, query string) ([]string, error) {
	home, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	var expanded string
	switch {
	case query == "", query == "~":
		expanded = home + string(filepath.Separator)
	case strings.HasPrefix(query, "~/"):
		expanded = home + string(filepath.Separator) + strings.TrimPrefix(query, "~/")
	case filepath.IsAbs(query):
		expanded = query
	default:
		return nil, errDirectoryOutsideHome
	}

	parent, prefix := expanded, ""
	if !strings.HasSuffix(query, string(filepath.Separator)) && query != "" && query != "~" {
		parent, prefix = filepath.Dir(expanded), filepath.Base(expanded)
	}
	parent = filepath.Clean(parent)
	if !pathWithinHome(home, parent) {
		if directories, isAncestor := suggestHomeAncestor(home, parent, prefix); isAncestor {
			return directories, nil
		}
		return nil, errDirectoryOutsideHome
	}
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return nil, err
	}
	if !pathWithinHome(resolvedHome, resolvedParent) {
		return nil, errDirectoryOutsideHome
	}

	entries, err := os.ReadDir(resolvedParent)
	if err != nil {
		return nil, err
	}
	lowerPrefix := strings.ToLower(prefix)
	showHidden := strings.HasPrefix(prefix, ".")
	directories := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if (!showHidden && strings.HasPrefix(name, ".")) || !strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
			continue
		}
		candidate := filepath.Join(parent, name)
		resolvedCandidate, resolveErr := filepath.EvalSymlinks(candidate)
		if resolveErr != nil || !pathWithinHome(resolvedHome, resolvedCandidate) {
			continue
		}
		info, statErr := os.Stat(candidate)
		if statErr != nil || !info.IsDir() {
			continue
		}
		directories = append(directories, candidate)
	}
	sort.Slice(directories, func(i, j int) bool {
		left, right := strings.ToLower(filepath.Base(directories[i])), strings.ToLower(filepath.Base(directories[j]))
		if left == right {
			return directories[i] < directories[j]
		}
		return left < right
	})
	if len(directories) > 20 {
		directories = directories[:20]
	}
	return directories, nil
}

// suggestHomeAncestor allows absolute-path completion to walk from the filesystem
// root toward home without listing any sibling directories along the way.
func suggestHomeAncestor(home, parent, prefix string) ([]string, bool) {
	rel, err := filepath.Rel(parent, home)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, false
	}
	nextPart := strings.Split(rel, string(filepath.Separator))[0]
	if !strings.HasPrefix(strings.ToLower(nextPart), strings.ToLower(prefix)) {
		return []string{}, true
	}
	return []string{filepath.Join(parent, nextPart)}, true
}

func pathWithinHome(home, path string) bool {
	rel, err := filepath.Rel(home, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		list := make([]model.Session, len(sessions))
		copy(list, sessions)
		mu.RUnlock()
		writeJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var req struct {
			Name        string         `json:"name"`
			WorkingDir  string         `json:"workingDir"`
			AgentType   string         `json:"agentType"`
			AgentConfig map[string]any `json:"agentConfig"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.AgentType == "" {
			http.Error(w, "name and agentType are required", http.StatusBadRequest)
			return
		}
		s := model.Session{
			ID:          uuid.NewString(),
			Name:        req.Name,
			WorkingDir:  req.WorkingDir,
			AgentType:   req.AgentType,
			AgentConfig: req.AgentConfig,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		}
		if err := validateSessionConnector(s); err != nil {
			extensionManager.Stop(s.ID)
			http.Error(w, fmt.Sprintf("agent unavailable: %v", err), http.StatusServiceUnavailable)
			return
		}
		mu.Lock()
		sessions = append(sessions, s)
		snapshot := snapshotData()
		mu.Unlock()
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save data after create: %v\n", err)
		}
		writeJSON(w, http.StatusCreated, s)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var all []AgentTypeInfo
	for _, def := range builtinAgentDefs {
		all = append(all, AgentTypeInfo{
			ID:          def.Name,
			Label:       def.Name,
			Description: def.Description,
			Harness:     def.Harness.Type,
			Sandbox:     def.Harness.Sandbox,
			IsBuiltin:   true,
			Available:   agent.CLIAvailable(def.Harness.Type),
		})
	}

	userDefs, err := store.LoadADLDefinitions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: load ADL definitions: %v\n", err)
	}
	for _, def := range userDefs {
		if def.Kind == "workflow" {
			continue // workflows are not selectable as session agent types
		}
		all = append(all, AgentTypeInfo{
			ID:          def.Name,
			Label:       def.Name,
			Description: def.Description,
			Harness:     def.Harness.Type,
			Sandbox:     def.Harness.Sandbox,
			IsBuiltin:   false,
			Available:   true,
		})
	}

	writeJSON(w, http.StatusOK, all)
}

// findADLDef looks up an ADL definition by name from builtins and user-defined definitions.
// It also handles legacy Session.AgentType strings (e.g. "claude-code", "adl:name").
func findADLDef(agentType string) (model.ADLDefinition, bool) {
	// Map legacy type strings to their ADL definition name.
	if mapped, ok := legacyAgentTypeNames[agentType]; ok {
		agentType = mapped
	}
	// Strip legacy "adl:" prefix used by old sessions.
	agentType = strings.TrimPrefix(agentType, "adl:")

	for _, def := range builtinAgentDefs {
		if def.Name == agentType {
			return def, true
		}
	}
	userDefs, _ := store.LoadADLDefinitions()
	for _, def := range userDefs {
		if def.Name == agentType {
			return def, true
		}
	}
	return model.ADLDefinition{}, false
}

func validateSessionConnector(s model.Session) error {
	if def, ok := findADLDef(s.AgentType); ok {
		switch def.Harness.Type {
		case "docker":
			_, err := extensionManager.GetAgent(s.ID, "docker", s.WorkingDir, map[string]any{
				"image":         def.Harness.Image,
				"containerPort": def.Harness.ContainerPort,
			})
			return err
		case "remote":
			_, err := extensionManager.GetAgent(s.ID, "remote", s.WorkingDir, map[string]any{
				"host": def.Harness.Host,
				"port": def.Harness.Port,
			})
			return err
		}
		return nil
	}
	switch s.AgentType {
	case "docker", "remote":
		_, err := extensionManager.GetAgent(s.ID, s.AgentType, s.WorkingDir, s.AgentConfig)
		return err
	default:
		return nil
	}
}

// sessionHarnessType returns the harness type for a session, used for history and cleanup.
func sessionHarnessType(session model.Session) string {
	if def, ok := findADLDef(session.AgentType); ok {
		return def.Harness.Type
	}
	// Legacy fallback.
	switch session.AgentType {
	case "pi", "docker-pi":
		return "pi"
	case "opencode", "docker-opencode":
		return "opencode"
	}
	return "claude-code"
}

// prewarmExtensionType returns the in-process harness key for a session,
// or "" if the session uses docker, remote, or an unsupported harness.
func prewarmExtensionType(sessionAgentType string) string {
	if def, ok := findADLDef(sessionAgentType); ok {
		h := def.Harness
		if h.Sandbox == "docker" || h.Type == "docker" || h.Type == "remote" {
			return ""
		}
		switch h.Type {
		case "claude-code", "":
			return "claude-code"
		case "pi":
			return "pi"
		case "codex":
			return "codex"
		case "opencode":
			return "opencode"
		default:
			return ""
		}
	}
	switch sessionAgentType {
	case "claude-code":
		return "claude-code"
	case "pi":
		return "pi"
	case "codex":
		return "codex"
	case "opencode":
		return "opencode"
	default:
		return ""
	}
}

func findSession(id string) (model.Session, bool) {
	for _, s := range sessions {
		if s.ID == id {
			return s, true
		}
	}
	return model.Session{}, false
}

func deleteSession(id string) bool {
	for i, s := range sessions {
		if s.ID == id {
			sessions = append(sessions[:i], sessions[i+1:]...)
			return true
		}
	}
	return false
}

func renameSession(id, name string) bool {
	for i, s := range sessions {
		if s.ID == id {
			sessions[i].Name = name
			return true
		}
	}
	return false
}

func handleSession(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	// Route /api/sessions/:id/<sub>
	if id, rest, ok := strings.Cut(path, "/"); ok {
		switch rest {
		case "messages":
			handleSessionMessages(w, r, id)
		case "chat":
			handleSessionChat(w, r, id)
		case "history":
			handleSessionHistory(w, r, id)
		case "ag-ui":
			handleSessionAGUI(w, r, id)
		default:
			http.NotFound(w, r)
		}
		return
	}

	id := path
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		s, ok := findSession(id)
		mu.RUnlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, s)

	case http.MethodPatch:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		mu.Lock()
		found := renameSession(id, name)
		var updated model.Session
		var snapshot store.Data
		if found {
			updated, _ = findSession(id)
			snapshot = snapshotData()
		}
		mu.Unlock()
		if !found {
			http.NotFound(w, r)
			return
		}
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save data after rename: %v\n", err)
		}
		writeJSON(w, http.StatusOK, updated)

	case http.MethodDelete:
		mu.Lock()
		var agentSessionID, workingDir, agentType string
		if s, ok := findSession(id); ok {
			agentSessionID = agentSessions[id]
			workingDir = s.WorkingDir
			agentType = s.AgentType
		}
		removed := deleteSession(id)
		if removed {
			delete(agentSessions, id)
			delete(sessionMessages, id)
		}
		var snapshot store.Data
		if removed {
			snapshot = snapshotData()
		}
		mu.Unlock()
		if !removed {
			http.NotFound(w, r)
			return
		}
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save data after delete: %v\n", err)
		}
		extensionManager.Stop(id)
		if agentSessionID != "" {
			var delErr error
			switch sessionHarnessType(model.Session{AgentType: agentType, WorkingDir: workingDir}) {
			case "pi":
				delErr = store.DeletePiSession(workingDir, agentSessionID)
			case "codex":
				delErr = store.DeleteCodexSession(workingDir, agentSessionID)
			case "opencode":
				delErr = store.DeleteOpenCodeSession(workingDir, agentSessionID)
			default:
				delErr = store.DeleteClaudeSession(workingDir, agentSessionID)
			}
			if delErr != nil {
				fmt.Fprintf(os.Stderr, "warn: delete session file: %v\n", delErr)
			}
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSessionMessages(w http.ResponseWriter, r *http.Request, sessionID string) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		msgs := sessionMessages[sessionID]
		if msgs == nil {
			msgs = []model.ChatMessage{}
		}
		mu.RUnlock()
		writeJSON(w, http.StatusOK, msgs)

	case http.MethodPut:
		var msgs []model.ChatMessage
		if err := json.NewDecoder(r.Body).Decode(&msgs); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		mu.Lock()
		sessionMessages[sessionID] = msgs
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s, err := store.LoadSettings()
		if err != nil {
			http.Error(w, "failed to load settings", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, s)

	case http.MethodPut:
		current, err := store.LoadSettings()
		if err != nil {
			http.Error(w, "failed to load settings", http.StatusInternalServerError)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&current); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if current.Theme != "light" && current.Theme != "dark" {
			http.Error(w, "theme must be 'light' or 'dark'", http.StatusBadRequest)
			return
		}
		if err := store.SaveSettings(current); err != nil {
			http.Error(w, "failed to save settings", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, current)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSessionChat(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	mu.RLock()
	session, ok := findSession(sessionID)
	agentSessionID := agentSessions[sessionID]
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	userMsg := model.ChatMessage{
		ID:        uuid.NewString(),
		Role:      "user",
		Content:   req.Message,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	mu.Lock()
	sessionMessages[sessionID] = append(sessionMessages[sessionID], userMsg)
	mu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	var ag agent.Agent
	var isADL bool
	if def, found := findADLDef(session.AgentType); found {
		ag = agent.NewADLAgent(def, session.ID, extensionManager)
		// Multi-step workflow definitions don't have a resumable agent session.
		isADL = len(def.Steps) > 0 || def.Kind == "workflow"
	} else {
		// Legacy fallback for "docker" and "remote" sessions that predate the ADL model.
		var err error
		ag, err = extensionManager.GetAgent(session.ID, session.AgentType, session.WorkingDir, session.AgentConfig)
		if err != nil {
			http.Error(w, fmt.Sprintf("agent unavailable: %v", err), http.StatusServiceUnavailable)
			return
		}
	}
	events := make(chan agent.Event, 64)

	go func() {
		defer close(events)
		runReq := agent.RunRequest{
			WorkingDir: session.WorkingDir,
			Message:    req.Message,
		}
		if !isADL {
			runReq.SessionID = agentSessionID
		}
		err := ag.Run(r.Context(), runReq, events)
		if err != nil && r.Context().Err() == nil {
			events <- agent.Event{Type: agent.EventError, Error: err.Error()}
		}
	}()

	var assistantContent strings.Builder
	var newAgentSessionID string

	for ev := range events {
		data, _ := json.Marshal(ev)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		switch ev.Type {
		case agent.EventText:
			assistantContent.WriteString(ev.Content)
		case agent.EventDone:
			newAgentSessionID = ev.SessionID
		}
	}

	if assistantContent.Len() > 0 {
		assistantMsg := model.ChatMessage{
			ID:        uuid.NewString(),
			Role:      "assistant",
			Content:   assistantContent.String(),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		mu.Lock()
		sessionMessages[sessionID] = append(sessionMessages[sessionID], assistantMsg)
		if newAgentSessionID != "" && !isADL {
			agentSessions[sessionID] = newAgentSessionID
		}
		snapshot := snapshotData()
		mu.Unlock()
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save session: %v\n", err)
		}
	}
}

func handleSessionHistory(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mu.RLock()
	session, ok := findSession(sessionID)
	agentSessionID := agentSessions[sessionID]
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	// Multi-step workflow sessions don't have a single agent history file.
	if def, ok := findADLDef(session.AgentType); ok && (len(def.Steps) > 0 || def.Kind == "workflow") {
		writeJSON(w, http.StatusOK, []model.ChatMessage{})
		return
	}
	var msgs []model.ChatMessage
	var err error
	switch sessionHarnessType(session) {
	case "pi":
		msgs, err = store.LoadPiHistory(session.WorkingDir, agentSessionID)
	case "codex":
		msgs, err = store.LoadCodexHistory(session.WorkingDir, agentSessionID)
	case "opencode":
		msgs, err = store.LoadOpenCodeHistory(session.WorkingDir, agentSessionID)
	default:
		msgs, err = store.LoadClaudeHistory(session.WorkingDir, agentSessionID)
	}
	if err != nil {
		http.Error(w, "failed to load history", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	caps := Capabilities{
		Sandbox: SandboxCapabilities{
			Bwrap: agent.GetBwrapStatus(),
		},
	}
	writeJSON(w, http.StatusOK, caps)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
