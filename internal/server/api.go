// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"nui/internal/agent"
	"nui/internal/agents"
	"nui/internal/devcontainer"
	"nui/internal/extensions"
	"nui/internal/hitl"
	"nui/internal/model"
	"nui/internal/skills"
	"nui/internal/store"

	"github.com/google/uuid"
)

// AgentTypeInfo is the API shape returned by GET /api/agent-types.
// ID equals the ADL definition id and is stored in Session.AgentType.
type AgentTypeInfo struct {
	ID                         string                      `json:"id"`
	Label                      string                      `json:"label"`
	Description                string                      `json:"description,omitempty"`
	Harness                    string                      `json:"harness"`                      // claude-code | pi | codex | opencode | api | docker | remote
	Provider                   string                      `json:"provider,omitempty"`           // harness type=api: anthropic | openai | gemini | ollama | openrouter
	Sandbox                    string                      `json:"sandbox,omitempty"`            // none | bubblewrap | docker
	PromptMode                 string                      `json:"promptMode,omitempty"`         // user | auto
	HitlMode                   string                      `json:"hitlMode,omitempty"`           // interactive | auto | off
	HarnessPermissions         string                      `json:"harnessPermissions,omitempty"` // interactive | bypass
	SupportsHarnessPermissions bool                        `json:"supportsHarnessPermissions,omitempty"`
	ToolApprovalPolicy         string                      `json:"toolApprovalPolicy,omitempty"` // default | all | allowlist | denylist
	ToolApprovalTools          []string                    `json:"toolApprovalTools,omitempty"`
	DefaultPrompt              string                      `json:"defaultPrompt,omitempty"`
	PromptSuggestions          []model.ADLPromptSuggestion `json:"promptSuggestions,omitempty"`
	Skills                     []string                    `json:"skills,omitempty"`
	WorkingDirInput            bool                        `json:"workingDirInput,omitempty"` // true = user picks working dir at session create
	Tags                       []string                    `json:"tags,omitempty"`
	AllowedHarnesses           []string                    `json:"allowedHarnesses,omitempty"` // effective CLI allowlist (all CLI types when omitted on a CLI agent)
	IsBuiltin                  bool                        `json:"isBuiltin"`
	Source                     string                      `json:"source,omitempty"` // builtin | user | extension
	Available                  bool                        `json:"available"`        // false when the required CLI is not installed
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
	backfillSessionsLastRunAt()
	return nil
}

// snapshotData must be called with mu held.
func snapshotData() store.Data {
	return snapshotDataFiltered()
}

func registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/sessions", handleSessions)
	mux.HandleFunc("/api/sessions/", handleSession)
	mux.HandleFunc("/api/agent-types", handleAgentTypes)
	mux.HandleFunc("/api/directories", handleDirectories)
	mux.HandleFunc("/api/settings", handleSettings)
	mux.HandleFunc("/api/state", handleState)
	mux.HandleFunc("/api/credentials", handleCredentials)
	mux.HandleFunc("/api/env", handleEnv)
	mux.HandleFunc("/api/bootstrap", handleBootstrap)
	mux.HandleFunc("/api/launch", handleLaunch)
	mux.HandleFunc("/api/orchestrate", handleOrchestrate)
	mux.HandleFunc("/api/orchestrator/routable-agents", handleOrchestratorRoutableAgents)
	mux.HandleFunc("/api/capabilities", handleCapabilities)
	mux.HandleFunc("/api/extensions", handleExtensions)
	mux.HandleFunc("/api/extensions/reload", handleExtensionsReload)
	mux.HandleFunc("/api/extensions/", handleExtensionPath)
	mux.HandleFunc("/api/mcp-servers", handleMCPServers)
	mux.HandleFunc("/api/skills", handleSkills)
	mux.HandleFunc("/api/skills/", handleSkill)
	mux.HandleFunc("/api/memory", handleMemory)
	mux.HandleFunc("/api/memory/", handleMemoryPath)
	mux.HandleFunc("/api/agents", handleAgents)
	mux.HandleFunc("/api/agents/", handleAgentFile)
	mux.HandleFunc("/api/agent-deployers", handleAgentDeployers)
	mux.HandleFunc("/api/schedules", handleSchedules)
	mux.HandleFunc("/api/schedules/", handleSchedule)
	registerHITLRoutes(mux)
	registerMCPOAuthRoutes(mux)
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
		writeJSON(w, http.StatusOK, enrichSessions(list))

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
		if req.AgentType == "" {
			http.Error(w, "agentType is required", http.StatusBadRequest)
			return
		}
		s, err := createSession(req.Name, req.WorkingDir, req.AgentType, req.AgentConfig)
		if err != nil {
			http.Error(w, fmt.Sprintf("agent unavailable: %v", err), http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusCreated, s)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleBulkDeleteSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.IDs) == 0 {
		http.Error(w, "ids is required", http.StatusBadRequest)
		return
	}

	type pendingDelete struct {
		id   string
		info sessionDeleteInfo
	}

	var toCleanup []pendingDelete
	deleted := make([]string, 0, len(req.IDs))
	notFound := make([]string, 0)

	mu.Lock()
	for _, id := range req.IDs {
		info, ok := sessionDeleteInfoFor(id)
		if !ok {
			notFound = append(notFound, id)
			continue
		}
		if purgeSessionFromMemory(id) {
			toCleanup = append(toCleanup, pendingDelete{id: id, info: info})
			deleted = append(deleted, id)
		}
	}
	var snapshot store.Data
	if len(deleted) > 0 {
		snapshot = snapshotData()
	}
	mu.Unlock()

	if len(deleted) > 0 {
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save data after bulk delete: %v\n", err)
		}
		for _, item := range toCleanup {
			cleanupDeletedSession(item.id, item.info)
		}
		notifySessionsChanged()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":  deleted,
		"notFound": notFound,
	})
}

func handleAgentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, listAgentTypes())
}

func handleOrchestratorRoutableAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, orchestratorAgentEntries(listAgentTypes()))
}

func agentTypeInfoFromDef(def model.ADLDefinition, builtin bool) AgentTypeInfo {
	info := AgentTypeInfo{
		ID:                model.ADLAgentID(def),
		Label:             model.ADLAgentLabel(def),
		Description:       def.Description,
		Harness:           def.Harness.Type,
		Provider:          def.Harness.Provider,
		Sandbox:           def.Harness.Sandbox,
		DefaultPrompt:     def.DefaultPrompt,
		PromptSuggestions: def.PromptSuggestions,
		Skills:            skillNamesFromADL(def),
		WorkingDirInput:   model.IsADLWorkingDirInput(def),
		Tags:              def.Tags,
		AllowedHarnesses:  effectiveAllowedHarnesses(def),
		IsBuiltin:         builtin,
		Available:         harnessAvailable(def),
	}
	if model.IsADLAutoPrompt(def) {
		info.PromptMode = model.ADLPromptModeAuto
	}
	mode := def.HITL.Mode
	if mode == "" {
		if model.IsADLAutoPrompt(def) {
			mode = hitl.ModeAuto
		} else {
			mode = hitl.ModeInteractive
		}
	}
	info.HitlMode = mode
	info.HarnessPermissions = hitl.EffectivePermissions(def, nil)
	info.SupportsHarnessPermissions = agent.HarnessSupportsUserScope(def.Harness.Type)
	info.ToolApprovalPolicy, info.ToolApprovalTools = hitl.EffectiveToolApprovals(def, nil)
	return info
}

func enrichExtensionAgentInfo(info *AgentTypeInfo) {
	if info == nil || extensions.Default == nil {
		return
	}
	extName, _, ok := extensions.ParseExtRef(info.ID)
	if !ok {
		return
	}
	ext, ok := extensions.Default.Get(extName)
	if !ok {
		return
	}
	displayName := strings.TrimSpace(ext.Manifest.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(ext.Manifest.Name)
	}
	if displayName == "" {
		return
	}
	info.Tags = appendUniqueTag(info.Tags, extensionDisplayTag(displayName))
}

func extensionDisplayTag(displayName string) string {
	displayName = strings.ToLower(displayName)
	displayName = strings.NewReplacer("&", " ", "'", "", ".", " ").Replace(displayName)
	parts := strings.Fields(displayName)
	return strings.Join(parts, "-")
}

func appendUniqueTag(tags []string, tag string) []string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return tags
	}
	for _, existing := range tags {
		if strings.EqualFold(existing, tag) {
			return tags
		}
	}
	return append(tags, tag)
}

func orchestratorAgentEntries(all []AgentTypeInfo) []map[string]any {
	candidates := orchestratorListableAgents(all)
	out := make([]map[string]any, 0, len(candidates))
	for _, a := range candidates {
		entry := map[string]any{
			"id":          a.ID,
			"label":       a.Label,
			"description": a.Description,
			"harness":     a.Harness,
			"tags":        a.Tags,
			"source":      a.Source,
			"launchable":  true,
		}
		if a.PromptMode != "" {
			entry["promptMode"] = a.PromptMode
		}
		if a.DefaultPrompt != "" {
			entry["defaultPrompt"] = a.DefaultPrompt
		}
		out = append(out, entry)
	}
	return out
}

func skillNamesFromADL(def model.ADLDefinition) []string {
	defCopy := def
	model.NormalizeADLSkills(&defCopy)

	seen := map[string]bool{}
	var names []string
	add := func(skills []model.ADLSkill) {
		for _, skill := range skills {
			name := strings.TrimSpace(skill.Name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	add(defCopy.AIAssets.Skills)
	for _, step := range defCopy.Steps {
		add(step.AIAssets.Skills)
	}
	for _, name := range skills.BuiltinSkillNames() {
		if name == skills.HitlAskUserSkillName || name == skills.RememberSkillName || seen[name] {
			continue
		}
		names = append(names, name)
	}
	return names
}

// harnessAvailable reports whether the harness required by def can run on this system.
func harnessAvailable(def model.ADLDefinition) bool {
	switch def.Harness.Type {
	case "claude-code", "pi", "codex", "opencode":
		return agent.CLIAvailable(def.Harness.Type)
	case "api":
		return agent.APIHarnessAvailable(def.Harness)
	case "devcontainer":
		// List in UI when the devcontainer CLI is installed; Docker is checked at runtime.
		return devcontainer.Available()
	default:
		return true
	}
}

// effectiveAllowedHarnesses returns the session-override allowlist, filtered to
// harnesses that are available on this system. Always includes the default when it is available.
func effectiveAllowedHarnesses(def model.ADLDefinition) []string {
	raw := model.NormalizeAllowedHarnesses(def)
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		probe := def
		probe.Harness.Type = t
		if harnessAvailable(probe) {
			out = append(out, t)
		}
	}
	return out
}

// findADLDef looks up an ADL definition by id from builtins and user-defined definitions.
// It also handles legacy Session.AgentType strings (harness names, old display names, "adl:id").
func findADLDef(agentType string) (model.ADLDefinition, bool) {
	return resolveAgentDefinition(agentType)
}

// resolveSessionADLDef returns the session's ADL definition with harness.type overridden
// from agentConfig.harnessType when allowed.
func resolveSessionADLDef(session model.Session) (model.ADLDefinition, bool) {
	def, ok := findADLDef(session.AgentType)
	if !ok {
		return model.ADLDefinition{}, false
	}
	effective, err := agents.ApplySessionHarnessOverride(def, session.AgentConfig)
	if err != nil {
		// Fall back to authored default if stored override is stale/invalid.
		return def, true
	}
	return effective, true
}

func validateSessionConnector(s model.Session) error {
	if def, ok := resolveSessionADLDef(s); ok {
		if !harnessAvailable(def) {
			return fmt.Errorf("%s harness is not available on this system", def.Harness.Type)
		}
		switch def.Harness.Type {
		case "docker":
			if strings.TrimSpace(def.Harness.Image) == "" {
				return fmt.Errorf("docker harness requires image")
			}
			if def.Harness.ContainerPort <= 0 {
				return fmt.Errorf("docker harness requires containerPort")
			}
		case "remote":
			if strings.TrimSpace(def.Harness.Host) == "" {
				return fmt.Errorf("remote harness requires host")
			}
			if def.Harness.Port <= 0 {
				return fmt.Errorf("remote harness requires port")
			}
		case "devcontainer":
			if strings.TrimSpace(def.Harness.InnerHarness) == "" {
				return fmt.Errorf("devcontainer harness requires innerHarness")
			}
			if !model.IsDevcontainerInnerHarness(def.Harness.InnerHarness) {
				return fmt.Errorf("devcontainer harness has unknown innerHarness %q", def.Harness.InnerHarness)
			}
		}
		return nil
	}
	switch s.AgentType {
	case "docker":
		if s.AgentConfig == nil {
			return fmt.Errorf("docker agent requires agentConfig")
		}
		image, _ := s.AgentConfig["image"].(string)
		if strings.TrimSpace(image) == "" {
			return fmt.Errorf("docker agent requires image in agentConfig")
		}
	case "remote":
		if s.AgentConfig == nil {
			return fmt.Errorf("remote agent requires agentConfig")
		}
		host, _ := s.AgentConfig["host"].(string)
		port, _ := s.AgentConfig["port"].(float64)
		if strings.TrimSpace(host) == "" || port <= 0 {
			return fmt.Errorf("remote agent requires host and port in agentConfig")
		}
	}
	return nil
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

type sessionDeleteInfo struct {
	agentSessionID string
	workingDir     string
	agentType      string
}

func sessionDeleteInfoFor(id string) (sessionDeleteInfo, bool) {
	s, ok := findSession(id)
	if !ok {
		return sessionDeleteInfo{}, false
	}
	return sessionDeleteInfo{
		agentSessionID: agentSessions[id],
		workingDir:     s.WorkingDir,
		agentType:      s.AgentType,
	}, true
}

func purgeSessionFromMemory(id string) bool {
	removed := deleteSession(id)
	if removed {
		delete(agentSessions, id)
		delete(sessionMessages, id)
	}
	return removed
}

func cleanupDeletedSession(id string, info sessionDeleteInfo) {
	purgeSessionRuns(id)
	if hitlCoord := coordinator(); hitlCoord != nil {
		if err := hitlCoord.DeleteSession(id); err != nil {
			fmt.Fprintf(os.Stderr, "warn: remove hitl requests: %v\n", err)
		}
	}
	if extensionManager != nil {
		extensionManager.Stop(id)
	}
	if err := store.RemoveSessionConfigDir(id); err != nil {
		fmt.Fprintf(os.Stderr, "warn: remove session config dir: %v\n", err)
	}
	if err := store.RemoveSessionWorkspace(id); err != nil {
		fmt.Fprintf(os.Stderr, "warn: remove session workspace: %v\n", err)
	}
	if err := store.RemoveSessionUploads(id); err != nil {
		fmt.Fprintf(os.Stderr, "warn: remove session uploads: %v\n", err)
	}
	clearLastSessionIDIfMatch(id)
	pruneRecentSessionID(id)
	session := model.Session{AgentType: info.agentType, WorkingDir: info.workingDir}
	if sessionUsesExtensionStorage(session) {
		deleteExtensionSession(id, session, info.workingDir, info.agentSessionID)
		return
	}
	if info.agentSessionID == "" {
		return
	}
	var delErr error
	switch sessionHarnessType(model.Session{AgentType: info.agentType, WorkingDir: info.workingDir}) {
	case "pi":
		delErr = store.DeletePiSession(info.workingDir, info.agentSessionID)
	case "codex":
		delErr = store.DeleteCodexSession(info.workingDir, info.agentSessionID)
	case "opencode":
		delErr = store.DeleteOpenCodeSession(info.workingDir, info.agentSessionID)
	default:
		delErr = store.DeleteClaudeSession(info.workingDir, info.agentSessionID)
	}
	if delErr != nil {
		fmt.Fprintf(os.Stderr, "warn: delete session file: %v\n", delErr)
	}
}

func clearLastSessionIDIfMatch(sessionID string) {
	if sessionID == "" {
		return
	}
	st, err := store.LoadState()
	if err != nil || st.LastSessionID != sessionID {
		return
	}
	st.LastSessionID = ""
	if err := store.SaveState(st); err != nil {
		fmt.Fprintf(os.Stderr, "warn: clear lastSessionId: %v\n", err)
	}
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

	if path == "events" {
		handleSessionEvents(w, r)
		return
	}

	if path == "ensure-default" {
		handleEnsureDefaultSession(w, r)
		return
	}

	if path == "bulk-delete" {
		handleBulkDeleteSessions(w, r)
		return
	}

	if path == "new" {
		handleNewSession(w, r)
		return
	}

	if path == "create" {
		handleNewSession(w, r)
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
		case "mentions":
			handleSessionMentions(w, r, id)
		case "uploads":
			handleSessionUploads(w, r, id)
		case "stop":
			handleSessionStop(w, r, id)
		default:
			if strings.HasPrefix(rest, "uploads/") {
				filename := strings.TrimPrefix(rest, "uploads/")
				handleSessionUploadServe(w, r, id, filename)
				return
			}
			if strings.HasPrefix(rest, "runs") || rest == "runs" {
				handleSessionRunsRouter(w, r, id, rest)
				return
			}
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
		writeJSON(w, http.StatusOK, enrichSession(s))

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
		notifySessionsChanged()
		writeJSON(w, http.StatusOK, updated)

	case http.MethodDelete:
		mu.Lock()
		info, found := sessionDeleteInfoFor(id)
		removed := false
		if found {
			removed = purgeSessionFromMemory(id)
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
		notifySessionsChanged()
		cleanupDeletedSession(id, info)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSessionMessages(w http.ResponseWriter, r *http.Request, sessionID string) {
	mu.RLock()
	session, ok := findSession(sessionID)
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	workingDir, _ := effectiveWorkingDir(session.WorkingDir)
	loadExtensionSessionMessages(sessionID, session, workingDir)

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
		snapshot := snapshotData()
		mu.Unlock()
		persistSessionState(sessionID, session, workingDir)
		if err := store.SaveData(snapshot); err != nil {
			http.Error(w, "failed to save messages", http.StatusInternalServerError)
			return
		}
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
		current, err := store.LoadUserSettings()
		if err != nil {
			http.Error(w, "failed to load settings", http.StatusInternalServerError)
			return
		}
		var patch store.Settings
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if patch.Theme != "" {
			if patch.Theme != "light" && patch.Theme != "dark" {
				http.Error(w, "theme must be 'light' or 'dark'", http.StatusBadRequest)
				return
			}
			current.Theme = patch.Theme
		}
		if patch.UITheme != "" {
			if !validUITheme(patch.UITheme) {
				http.Error(w, "uiTheme must be 'hawaiian' or 'standard'", http.StatusBadRequest)
				return
			}
			current.UITheme = patch.UITheme
		}
		if patch.DisableSloganAnimation != nil {
			current.DisableSloganAnimation = patch.DisableSloganAnimation
		}
		if patch.DefaultAgentType != "" {
			current.DefaultAgentType = patch.DefaultAgentType
		}
		if patch.DefaultHarness != "" {
			if !agents.HarnessAvailable(patch.DefaultHarness) {
				http.Error(w, fmt.Sprintf("harness %q is not available on this system", patch.DefaultHarness), http.StatusBadRequest)
				return
			}
			current.DefaultHarness = patch.DefaultHarness
		}
		if patch.DisabledExtensions != nil {
			current.DisabledExtensions = patch.DisabledExtensions
		}
		if patch.MCPOAuthCallbackURL != "" {
			current.MCPOAuthCallbackURL = patch.MCPOAuthCallbackURL
		}
		if patch.MemoryUserMode != "" {
			current.MemoryUserMode = patch.MemoryUserMode
		}
		if patch.MemoryAgentsMode != nil {
			if current.MemoryAgentsMode == nil {
				current.MemoryAgentsMode = map[string]string{}
			}
			for id, mode := range patch.MemoryAgentsMode {
				current.MemoryAgentsMode[id] = mode
			}
		}
		if patch.AutoCheckUpdates != nil {
			current.AutoCheckUpdates = patch.AutoCheckUpdates
		}
		if patch.UpdateCheckIntervalHours > 0 {
			current.UpdateCheckIntervalHours = patch.UpdateCheckIntervalHours
		}
		if patch.SkippedUpdateVersion != "" {
			current.SkippedUpdateVersion = patch.SkippedUpdateVersion
		}
		// Allow clearing skipped version with explicit empty via sentinel — use "-" to clear.
		if patch.SkippedUpdateVersion == "-" {
			current.SkippedUpdateVersion = ""
		}
		if current.Theme == "" {
			current.Theme = "light"
		}
		if current.UITheme == "" {
			current.UITheme = "hawaiian"
		}
		if err := store.SaveSettings(current); err != nil {
			http.Error(w, "failed to save settings", http.StatusInternalServerError)
			return
		}
		effective, err := store.LoadSettings()
		if err != nil {
			writeJSON(w, http.StatusOK, current)
			return
		}
		writeJSON(w, http.StatusOK, effective)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleState(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		st, err := store.LoadState()
		if err != nil {
			http.Error(w, "failed to load state", http.StatusInternalServerError)
			return
		}
		if populated, changed := ensureRecentsPopulated(st); changed {
			if err := store.SaveState(populated); err != nil {
				fmt.Fprintf(os.Stderr, "warn: save populated recents: %v\n", err)
			}
			st = populated
		}
		writeJSON(w, http.StatusOK, st)

	case http.MethodPut:
		current, err := store.LoadState()
		if err != nil {
			http.Error(w, "failed to load state", http.StatusInternalServerError)
			return
		}
		var patch store.State
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if patch.LastAgentType != "" {
			current.LastAgentType = patch.LastAgentType
		}
		if patch.LastSessionID != "" {
			current.LastSessionID = patch.LastSessionID
		}
		if patch.SidebarOpen != nil {
			current.SidebarOpen = patch.SidebarOpen
		}
		if patch.RecentsOpen != nil {
			current.RecentsOpen = patch.RecentsOpen
		}
		if patch.SidebarWidth != nil {
			w := *patch.SidebarWidth
			if w < 200 {
				w = 200
			}
			if w > 480 {
				w = 480
			}
			current.SidebarWidth = &w
		}
		if patch.RecentSessionIDs != nil {
			current.RecentSessionIDs = patch.RecentSessionIDs
		}
		if patch.RecentAgents != nil {
			current.RecentAgents = patch.RecentAgents
		}
		if err := store.SaveState(current); err != nil {
			http.Error(w, "failed to save state", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, current)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func validUITheme(id string) bool {
	switch id {
	case "hawaiian", "standard":
		return true
	default:
		return false
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

	appendUserMessage(sessionID, req.Message)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	runID := uuid.NewString()
	runCtx, cancelRun := context.WithCancel(r.Context())
	registerActiveRun(sessionID, runID, cancelRun)
	defer unregisterActiveRun(sessionID, runID)

	result := executeRun(runCtx, executeRunOptions{
		SessionID:      sessionID,
		Session:        session,
		Message:        req.Message,
		RunID:          runID,
		PersistLog:     true,
		AgentSessionID: agentSessionID,
		OnEvent: func(ev agent.Event) {
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		},
	})

	if result.AssistantContent != "" {
		persistAssistantTurn(sessionID, result.AssistantContent, result.NewAgentSessionID, sessionSkipsTopLevelHarnessSession(session))
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
	workingDir, err := effectiveWorkingDir(session.WorkingDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Multi-step workflow sessions don't have a single agent history file.
	if def, ok := findADLDef(session.AgentType); ok && model.IsMultiStepWorkflow(def) {
		writeJSON(w, http.StatusOK, []model.ChatMessage{})
		return
	}
	var msgs []model.ChatMessage
	switch sessionHarnessType(session) {
	case "pi":
		msgs, err = store.LoadPiHistory(workingDir, agentSessionID)
	case "codex":
		msgs, err = store.LoadCodexHistory(workingDir, agentSessionID)
	case "opencode":
		msgs, err = store.LoadOpenCodeHistory(workingDir, agentSessionID)
	default:
		msgs, err = store.LoadClaudeHistory(workingDir, agentSessionID)
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
