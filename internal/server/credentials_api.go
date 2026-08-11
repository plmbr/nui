// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"nui/internal/agent"
	"nui/internal/store"
)

// CredentialField is the API shape for one managed credential env var.
type CredentialField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Group       string `json:"group"`
	Secret      bool   `json:"secret"`
	Value       string `json:"value"`                // value from ~/.nui/secrets.json only
	FromEnv     bool   `json:"fromEnv"`              // process environment currently has a value
	Configured  bool   `json:"configured"`           // available via secrets, process env, or both
}

type credentialsResponse struct {
	Fields []CredentialField `json:"fields"`
}

type credentialsPatch struct {
	Env map[string]string `json:"env"`
}

func handleCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, credentialsResponse{Fields: buildCredentialFields()})

	case http.MethodPut:
		var patch credentialsPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if patch.Env == nil {
			http.Error(w, "env is required", http.StatusBadRequest)
			return
		}
		for key := range patch.Env {
			if !agent.IsManagedCredentialKey(key) {
				http.Error(w, "unknown credential key: "+key, http.StatusBadRequest)
				return
			}
		}
		current, err := store.LoadSecrets()
		if err != nil {
			http.Error(w, "failed to load secrets", http.StatusInternalServerError)
			return
		}
		if current.Env == nil {
			current.Env = map[string]string{}
		}
		for key, value := range patch.Env {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if value == "" {
				delete(current.Env, key)
			} else {
				current.Env[key] = value
			}
		}
		if err := store.SaveSecrets(current); err != nil {
			http.Error(w, "failed to save secrets", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, credentialsResponse{Fields: buildCredentialFields()})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func buildCredentialFields() []CredentialField {
	secrets, _ := store.LoadSecrets()
	if secrets.Env == nil {
		secrets.Env = map[string]string{}
	}
	specs := agent.CredentialFieldSpecs()
	out := make([]CredentialField, 0, len(specs))
	for _, spec := range specs {
		stored := strings.TrimSpace(secrets.Env[spec.Key])
		fromEnv := strings.TrimSpace(os.Getenv(spec.Key)) != ""
		out = append(out, CredentialField{
			Key:         spec.Key,
			Label:       spec.Label,
			Description: spec.Description,
			Group:       spec.Group,
			Secret:      spec.Secret,
			Value:       stored,
			FromEnv:     fromEnv,
			Configured:  stored != "" || fromEnv,
		})
	}
	return out
}
