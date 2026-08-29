// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"nui/internal/model"
)

type RecentAgentEntry struct {
	AgentType   string         `json:"agentType"`
	WorkingDir  string         `json:"workingDir,omitempty"`
	AgentConfig map[string]any `json:"agentConfig,omitempty"`
	UsedAt      string         `json:"usedAt,omitempty"`
}

// Settings holds durable preferences (admin-seedable; user overrides).
type Settings struct {
	Theme                    string            `json:"theme"`                              // "light" | "dark" color mode
	UITheme                  string            `json:"uiTheme,omitempty"`                  // visual theme id, e.g. "hawaiian" | "standard"; default hawaiian
	DisableSloganAnimation   *bool             `json:"disableSloganAnimation,omitempty"`   // skip landing-page slogan word animation
	DefaultAgentType         string            `json:"defaultAgentType,omitempty"`         // default agent for new sessions on launch
	DefaultHarness           string            `json:"defaultHarness,omitempty"`           // harness for internal agents (e.g. api/anthropic, claude-code)
	DisabledExtensions       []string          `json:"disabledExtensions,omitempty"`       // extension names excluded from runtime
	MCPOAuthCallbackURL      string            `json:"mcpOAuthCallbackUrl,omitempty"`      // optional OAuth callback base URL override
	MemoryUserMode           string            `json:"memoryUserMode,omitempty"`           // auto | manual | disabled; default manual
	MemoryAgentsMode         map[string]string `json:"memoryAgentsMode,omitempty"`         // per ADL agent id; missing = manual
	AutoCheckUpdates         *bool             `json:"autoCheckUpdates,omitempty"`         // nil/true = check periodically; false = off
	UpdateCheckIntervalHours int               `json:"updateCheckIntervalHours,omitempty"` // default 24
	SkippedUpdateVersion     string            `json:"skippedUpdateVersion,omitempty"`     // dismiss banner until a newer version
}

// AutoCheckUpdatesEnabled reports whether periodic update checks are on (default true).
func AutoCheckUpdatesEnabled(s Settings) bool {
	if s.AutoCheckUpdates == nil {
		return true
	}
	return *s.AutoCheckUpdates
}

// UpdateCheckInterval returns the check interval, defaulting to 24 hours.
func UpdateCheckInterval(s Settings) int {
	if s.UpdateCheckIntervalHours <= 0 {
		return 24
	}
	return s.UpdateCheckIntervalHours
}

// State holds ephemeral UI restoration data (user-only; never admin-seeded).
type State struct {
	LastAgentType    string             `json:"lastAgentType,omitempty"` // last agent picked in new-session dialog
	LastSessionID    string             `json:"lastSessionId,omitempty"` // last selected session in UI
	RecentSessionIDs []string           `json:"recentSessionIds,omitempty"`
	RecentAgents     []RecentAgentEntry `json:"recentAgents,omitempty"`
	RecentsOpen      *bool              `json:"recentsOpen,omitempty"`  // Recents section expanded on launch/new-session
	SidebarOpen      *bool              `json:"sidebarOpen,omitempty"`  // desktop sidebar expanded state
	SidebarWidth     *int               `json:"sidebarWidth,omitempty"` // desktop sidebar width in px
}

type Data struct {
	Sessions        []model.Session                `json:"sessions"`
	AgentSessions   map[string]string              `json:"agentSessions"`
	SessionMessages map[string][]model.ChatMessage `json:"sessionMessages,omitempty"`
}

func defaultSettings() Settings {
	return Settings{Theme: "light", UITheme: "hawaiian"}
}

func normalizeSettings(s *Settings) {
	if s.Theme == "" {
		s.Theme = "light"
	}
	if s.UITheme == "" {
		s.UITheme = "hawaiian"
	}
}

func loadSettingsFile(dir string) (Settings, error) {
	defaults := defaultSettings()
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return defaults, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return defaults, err
	}
	normalizeSettings(&s)
	return s, nil
}

// LoadUserSettings reads settings.json from the user data dir only.
func LoadUserSettings() (Settings, error) {
	dir, err := UserDir()
	if err != nil {
		return defaultSettings(), err
	}
	return loadSettingsFile(dir)
}

// LoadSystemSettings reads settings.json from the system config dir (if present).
func LoadSystemSettings() (Settings, error) {
	if !SystemDirExists() {
		return defaultSettings(), nil
	}
	return loadSettingsFile(SystemDir())
}

// LoadSettings returns effective settings: system base merged with user overrides.
func LoadSettings() (Settings, error) {
	sys, err := LoadSystemSettings()
	if err != nil {
		sys = defaultSettings()
	}
	user, err := LoadUserSettings()
	if err != nil {
		return mergeSettings(sys, defaultSettings()), err
	}
	return mergeSettings(sys, user), nil
}

// SaveSettings writes preferences to the user data dir only.
func SaveSettings(s Settings) error {
	normalizeSettings(&s)
	return saveJSON("settings.json", s)
}

func LoadState() (State, error) {
	empty := State{}
	dir, err := UserDir()
	if err != nil {
		return empty, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return empty, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return empty, err
	}
	return st, nil
}

func SaveState(st State) error {
	return saveJSON("state.json", st)
}

func LoadData() (Data, error) {
	empty := Data{
		Sessions:        []model.Session{},
		AgentSessions:   map[string]string{},
		SessionMessages: map[string][]model.ChatMessage{},
	}
	dir, err := UserDir()
	if err != nil {
		return empty, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "data.json"))
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return empty, err
	}
	var d Data
	if err := json.Unmarshal(raw, &d); err != nil {
		return empty, err
	}
	if d.Sessions == nil {
		d.Sessions = []model.Session{}
	}
	if d.AgentSessions == nil {
		d.AgentSessions = map[string]string{}
	}
	if d.SessionMessages == nil {
		d.SessionMessages = map[string][]model.ChatMessage{}
	}
	return d, nil
}

func SaveData(d Data) error {
	return saveJSON("data.json", d)
}

func AgentsDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	agentsDir := filepath.Join(dir, "agents")
	if err := os.MkdirAll(agentsDir, 0700); err != nil {
		return "", err
	}
	return agentsDir, nil
}

func ExtensionsDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	extDir := filepath.Join(dir, "extensions")
	if err := os.MkdirAll(extDir, 0700); err != nil {
		return "", err
	}
	return extDir, nil
}

// ConnectionsDir returns ~/.nui/connections where harness TCP/HTTP handshake files are written.
func ConnectionsDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	connDir := filepath.Join(dir, "connections")
	if err := os.MkdirAll(connDir, 0700); err != nil {
		return "", err
	}
	return connDir, nil
}

// ConnectionFilePath returns ~/.nui/connections/<connectionID>.json.
func ConnectionFilePath(connectionID string) (string, error) {
	dir, err := ConnectionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, connectionID+".json"), nil
}

func loadADLDefinitionsFromDir(dir string) []model.ADLDefinition {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var defs []model.ADLDefinition
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var def model.ADLDefinition
		if err := yaml.Unmarshal(raw, &def); err != nil {
			continue
		}
		model.NormalizeADLDefinition(&def)
		model.NormalizeADLSkills(&def)
		if def.ID == "" && def.Name == "" {
			continue
		}
		if err := model.ValidateADLDefinition(def); err != nil {
			fmt.Fprintf(os.Stderr, "warn: skip invalid ADL %q: %v\n", e.Name(), err)
			continue
		}
		defs = append(defs, def)
	}
	return defs
}

// LoadADLDefinitions returns system + user ADL agents (user wins on same id).
func LoadADLDefinitions() ([]model.ADLDefinition, error) {
	byID := map[string]model.ADLDefinition{}
	order := []string{}

	add := func(defs []model.ADLDefinition) {
		for _, def := range defs {
			id := model.ADLAgentID(def)
			if id == "" {
				continue
			}
			if _, exists := byID[id]; !exists {
				order = append(order, id)
			}
			byID[id] = def
		}
	}

	if SystemDirExists() {
		add(loadADLDefinitionsFromDir(filepath.Join(SystemDir(), "agents")))
	}
	userAgents, err := AgentsDir()
	if err != nil {
		return nil, err
	}
	add(loadADLDefinitionsFromDir(userAgents))

	out := make([]model.ADLDefinition, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out, nil
}

func saveJSON(filename string, v any) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_, werr := tmp.Write(b)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	if werr != nil {
		os.Remove(tmpPath)
		return werr
	}
	if syncErr != nil {
		os.Remove(tmpPath)
		return syncErr
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return closeErr
	}
	return os.Rename(tmpPath, filepath.Join(dir, filename))
}
