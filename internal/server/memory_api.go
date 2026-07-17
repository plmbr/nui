// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"nui/internal/memory"
	"nui/internal/storageext"
	"nui/internal/store"
)

func handleMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	settings, err := store.LoadSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	summary, err := memoryListSummary(settings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func handleMemoryPath(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/memory/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		http.Error(w, "invalid memory path", http.StatusBadRequest)
		return
	}

	if rest == "user" {
		handleUserMemory(w, r)
		return
	}

	const agentsPrefix = "agents/"
	if strings.HasPrefix(rest, agentsPrefix) {
		agentID := strings.TrimPrefix(rest, agentsPrefix)
		if agentID == "" || strings.Contains(agentID, "/") || strings.Contains(agentID, "..") {
			http.Error(w, "invalid agent id", http.StatusBadRequest)
			return
		}
		handleAgentMemory(w, r, agentID)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func handleUserMemory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		content, err := memory.ReadUser()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"content": content})

	case http.MethodPut:
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := memory.WriteUser(req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		if err := memory.DeleteUser(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgentMemory(w http.ResponseWriter, r *http.Request, agentID string) {
	switch r.Method {
	case http.MethodGet:
		content, err := memory.ReadAgent(agentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"agentId": agentID,
			"content": content,
		})

	case http.MethodPut:
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := memory.WriteAgent(agentID, req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func memoryListSummary(settings store.Settings) (memory.Summary, error) {
	if storageext.Default != nil {
		return storageext.Default.ListSummary(settings)
	}
	return memory.ListSummary(settings)
}
