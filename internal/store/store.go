// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"loop/internal/model"
)

type Settings struct {
	Theme                string   `json:"theme"`                            // "light" | "dark"
	DefaultAgentType     string   `json:"defaultAgentType,omitempty"`       // default agent for new sessions on launch
	LastAgentType        string   `json:"lastAgentType,omitempty"`          // last agent picked in new-session dialog
	LastSessionID        string   `json:"lastSessionId,omitempty"`          // last selected session in UI
	SidebarOpen          *bool    `json:"sidebarOpen,omitempty"`            // desktop sidebar expanded state
	SidebarWidth         *int     `json:"sidebarWidth,omitempty"`           // desktop sidebar width in px
	DisabledExtensions   []string `json:"disabledExtensions,omitempty"`     // extension names excluded from runtime
}

type Data struct {
	Sessions        []model.Session              `json:"sessions"`
	AgentSessions   map[string]string            `json:"agentSessions"`
	SessionMessages map[string][]model.ChatMessage `json:"sessionMessages,omitempty"`
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".loop")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func LoadSettings() (Settings, error) {
	dir, err := Dir()
	if err != nil {
		return Settings{Theme: "light"}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Settings{Theme: "light"}, nil
	}
	if err != nil {
		return Settings{Theme: "light"}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{Theme: "light"}, err
	}
	if s.Theme == "" {
		s.Theme = "light"
	}
	return s, nil
}

func SaveSettings(s Settings) error {
	return saveJSON("settings.json", s)
}

func LoadData() (Data, error) {
	empty := Data{
		Sessions:        []model.Session{},
		AgentSessions:   map[string]string{},
		SessionMessages: map[string][]model.ChatMessage{},
	}
	dir, err := Dir()
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
	// Parse into raw fields first to handle both old and new formats.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return empty, err
	}
	d := Data{
		Sessions:        []model.Session{},
		AgentSessions:   map[string]string{},
		SessionMessages: map[string][]model.ChatMessage{},
	}

	// New format: "sessions" is an array of Session objects.
	if sessRaw, ok := fields["sessions"]; ok {
		json.Unmarshal(sessRaw, &d.Sessions) //nolint:errcheck — falls back to legacy if wrong type
	}
	if asRaw, ok := fields["agentSessions"]; ok {
		json.Unmarshal(asRaw, &d.AgentSessions) //nolint:errcheck
	}
	if smRaw, ok := fields["sessionMessages"]; ok {
		json.Unmarshal(smRaw, &d.SessionMessages) //nolint:errcheck
	}

	// Legacy migration: old format had "projects":[...] and "sessions":{map}.
	if len(d.Sessions) == 0 {
		if projRaw, ok := fields["projects"]; ok {
			json.Unmarshal(projRaw, &d.Sessions) //nolint:errcheck
		}
		if len(d.AgentSessions) == 0 {
			if sessRaw, ok := fields["sessions"]; ok {
				json.Unmarshal(sessRaw, &d.AgentSessions) //nolint:errcheck
			}
		}
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

// ConnectionsDir returns ~/.loop/connections where harness TCP/HTTP handshake files are written.
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

// ConnectionFilePath returns ~/.loop/connections/<connectionID>.json.
func ConnectionFilePath(connectionID string) (string, error) {
	dir, err := ConnectionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, connectionID+".json"), nil
}

func LoadADLDefinitions() ([]model.ADLDefinition, error) {
	dir, err := AgentsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
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
	return defs, nil
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
