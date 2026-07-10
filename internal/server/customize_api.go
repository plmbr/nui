// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"loop/internal/model"
	"loop/internal/skills"
	"loop/internal/store"

	"gopkg.in/yaml.v3"
)

type agentFileInfo struct {
	File        string `json:"file"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func handleMCPServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		servers, err := store.LoadMCPServers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"mcpServers": servers})

	case http.MethodPut:
		var body struct {
			MCPServers []model.ADLMCPServer `json:"mcpServers"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := store.SaveMCPServers(body.MCPServers); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"mcpServers": body.MCPServers})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := skills.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []skills.Entry{}
	}
	writeJSON(w, http.StatusOK, list)
}

func handleSkill(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/skills/")
	name = strings.Trim(name, "/")
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		http.Error(w, "invalid skill name", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := skills.Remove(name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := listAgentFiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var req struct {
			File    string `json:"file"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		filename, err := sanitizeAgentFilename(req.File)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			http.Error(w, "content is required", http.StatusBadRequest)
			return
		}
		if err := validateAgentYAMLContent(req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path, err := agentFilePath(filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := os.Stat(path); err == nil {
			http.Error(w, "agent file already exists", http.StatusConflict)
			return
		}
		if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		info, err := agentFileInfoFromPath(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, info)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgentFile(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		http.Error(w, "invalid agent file", http.StatusBadRequest)
		return
	}
	if strings.HasSuffix(rest, "/evals/run") {
		agentID := strings.TrimSuffix(rest, "/evals/run")
		agentID = strings.Trim(agentID, "/")
		if agentID == "" || strings.Contains(agentID, "/") {
			http.Error(w, "invalid agent id", http.StatusBadRequest)
			return
		}
		handleAgentEvalRun(w, r, agentID)
		return
	}
	if strings.HasSuffix(rest, "/deploy") {
		agentID := strings.TrimSuffix(rest, "/deploy")
		agentID = strings.Trim(agentID, "/")
		if agentID == "" || strings.Contains(agentID, "/") {
			http.Error(w, "invalid agent id", http.StatusBadRequest)
			return
		}
		handleAgentDeploy(w, r, agentID)
		return
	}
	filename := rest
	if strings.Contains(filename, "/") {
		http.Error(w, "invalid agent file", http.StatusBadRequest)
		return
	}
	safeName, err := sanitizeAgentFilename(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path, err := agentFilePath(safeName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"file":    safeName,
			"content": string(data),
		})

	case http.MethodPut:
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			http.Error(w, "content is required", http.StatusBadRequest)
			return
		}
		if err := validateAgentYAMLContent(req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		info, err := agentFileInfoFromPath(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, info)

	case http.MethodDelete:
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func listAgentFiles() ([]agentFileInfo, error) {
	dir, err := store.AgentsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []agentFileInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		info, err := agentFileInfoFromPath(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, info)
	}
	if out == nil {
		out = []agentFileInfo{}
	}
	return out, nil
}

func agentFilePath(filename string) (string, error) {
	dir, err := store.AgentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filename), nil
}

func sanitizeAgentFilename(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errInvalidAgentFile
	}
	if filepath.Base(name) != name || strings.Contains(name, "..") {
		return "", errInvalidAgentFile
	}
	ext := filepath.Ext(name)
	if ext != ".yaml" && ext != ".yml" {
		name += ".yaml"
	}
	return name, nil
}

var errInvalidAgentFile = &agentFileError{"invalid agent filename"}

type agentFileError struct{ msg string }

func (e *agentFileError) Error() string { return e.msg }

func validateAgentYAMLContent(content string) error {
	var def model.ADLDefinition
	if err := yaml.Unmarshal([]byte(content), &def); err != nil {
		return fmt.Errorf("parse agent ADL: %w", err)
	}
	model.NormalizeADLDefinition(&def)
	model.NormalizeADLSkills(&def)
	return model.ValidateADLDefinition(def)
}

func agentFileInfoFromPath(path string) (agentFileInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return agentFileInfo{}, err
	}
	var def model.ADLDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return agentFileInfo{}, err
	}
	model.NormalizeADLDefinition(&def)
	name := def.Name
	if name == "" {
		name = def.ID
	}
	return agentFileInfo{
		File:        filepath.Base(path),
		ID:          def.ID,
		Name:        name,
		Description: def.Description,
	}, nil
}
