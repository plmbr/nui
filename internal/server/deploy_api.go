// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"

	"loop/internal/agents"
	"loop/internal/extensions"
)

func handleAgentDeployers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	reg := extensions.Default
	if reg == nil {
		var err error
		reg, err = extensions.LoadRegistry()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	list := reg.AllDeployers()
	if list == nil {
		list = []extensions.DeployerInfo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"deployers": list})
}

func handleAgentDeploy(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		DeployerID string `json:"deployerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.DeployerID == "" {
		http.Error(w, "deployerId is required", http.StatusBadRequest)
		return
	}
	result, err := agents.Deploy(body.DeployerID, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
