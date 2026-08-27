// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
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
	Value       string `json:"value"`      // value from ~/.nui/secrets.json only
	FromEnv     bool   `json:"fromEnv"`    // process environment currently has a value
	Configured  bool   `json:"configured"` // available via secrets, process env, or both
}

// CustomEnvEntry is one free-form global env var from secrets.json.
type CustomEnvEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type envResponse struct {
	Fields []CredentialField `json:"fields"`
	Custom []CustomEnvEntry  `json:"custom"`
}

// credentialsResponse is kept for older clients that only read fields.
type credentialsResponse = envResponse

type envPatch struct {
	Env    map[string]string `json:"env"`
	Custom map[string]string `json:"custom"`
}

type credentialsPatch = envPatch

func handleCredentials(w http.ResponseWriter, r *http.Request) {
	handleEnv(w, r)
}

func handleEnv(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, buildEnvResponse())

	case http.MethodPut:
		var patch envPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if patch.Env == nil && patch.Custom == nil {
			http.Error(w, "env or custom is required", http.StatusBadRequest)
			return
		}
		for key := range patch.Env {
			if !agent.IsManagedCredentialKey(key) {
				http.Error(w, "unknown credential key: "+key, http.StatusBadRequest)
				return
			}
		}
		for key := range patch.Custom {
			key = strings.TrimSpace(key)
			if key == "" {
				http.Error(w, "custom env key is required", http.StatusBadRequest)
				return
			}
			if store.IsReservedEnvKey(key) {
				http.Error(w, "reserved env key: "+key, http.StatusBadRequest)
				return
			}
			if agent.IsManagedCredentialKey(key) {
				http.Error(w, "managed credential key must use env field: "+key, http.StatusBadRequest)
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
		if patch.Env != nil {
			applyEnvPatch(current.Env, patch.Env)
		}
		if patch.Custom != nil {
			for key := range current.Env {
				if !agent.IsManagedCredentialKey(key) {
					delete(current.Env, key)
				}
			}
			applyEnvPatch(current.Env, patch.Custom)
		}
		if err := store.SaveSecrets(current); err != nil {
			http.Error(w, "failed to save secrets", http.StatusInternalServerError)
			return
		}
		store.ApplyGlobalEnvToProcess()
		reloadExtensionRegistryAsync()
		writeJSON(w, http.StatusOK, buildEnvResponse())

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func applyEnvPatch(dst map[string]string, patch map[string]string) {
	for key, value := range patch {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if value == "" {
			delete(dst, key)
		} else {
			dst[key] = value
		}
	}
}

func buildEnvResponse() envResponse {
	return envResponse{
		Fields: buildCredentialFields(),
		Custom: buildCustomEnvEntries(),
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

func buildCustomEnvEntries() []CustomEnvEntry {
	secrets, _ := store.LoadSecrets()
	if secrets.Env == nil {
		return []CustomEnvEntry{}
	}
	keys := make([]string, 0)
	for key := range secrets.Env {
		if agent.IsManagedCredentialKey(key) {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]CustomEnvEntry, 0, len(keys))
	for _, key := range keys {
		out = append(out, CustomEnvEntry{Key: key, Value: secrets.Env[key]})
	}
	return out
}
