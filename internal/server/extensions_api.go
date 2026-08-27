// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"nui/internal/agent"
	"nui/internal/extensions"
	"nui/internal/mentions"
	"nui/internal/store"
)

func handleExtensions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if extensions.Default == nil {
		writeJSON(w, http.StatusOK, []extensions.ExtensionInfo{})
		return
	}
	writeJSON(w, http.StatusOK, extensions.Default.Info())
}

func handleExtensionsReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := reloadExtensionRegistry(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(os.Stderr, "[extensions] reload complete")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func handleExtensionPath(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/extensions/")
	rest = strings.Trim(rest, "/")
	if rest == "" || rest == "reload" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(rest, "/env") {
		name := strings.TrimSuffix(rest, "/env")
		name = strings.Trim(name, "/")
		handleExtensionEnv(w, r, name)
		return
	}
	http.NotFound(w, r)
}

type extensionEnvResponse struct {
	Keys []string          `json:"keys"`
	Env  map[string]string `json:"env,omitempty"`
}

type extensionEnvPatch struct {
	Env map[string]string `json:"env"`
}

func handleExtensionEnv(w http.ResponseWriter, r *http.Request, name string) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") {
		http.Error(w, "invalid extension name", http.StatusBadRequest)
		return
	}
	if !extensionInstalled(name) {
		http.Error(w, "extension not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		env := store.ExtensionEnv(name)
		writeJSON(w, http.StatusOK, extensionEnvResponse{
			Keys: store.ExtensionEnvKeys(name),
			Env:  env,
		})

	case http.MethodPut:
		var patch extensionEnvPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if patch.Env == nil {
			http.Error(w, "env is required", http.StatusBadRequest)
			return
		}
		for key := range patch.Env {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if store.IsReservedEnvKey(key) {
				http.Error(w, "reserved env key: "+key, http.StatusBadRequest)
				return
			}
			if agent.IsManagedCredentialKey(key) {
				// Allow managed keys on per-extension env (override globals for that extension).
				continue
			}
		}
		if err := store.SetExtensionEnv(name, patch.Env); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		keys := store.ExtensionEnvKeys(name)
		env := store.ExtensionEnv(name)
		reloadExtensionRegistryAsync()
		writeJSON(w, http.StatusOK, extensionEnvResponse{
			Keys: keys,
			Env:  env,
		})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func extensionInstalled(name string) bool {
	if extensions.Default != nil {
		for _, info := range extensions.Default.Info() {
			if info.Name == name {
				return true
			}
		}
	}
	extDir, err := store.ExtensionsDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(extDir, name))
	return err == nil && info.IsDir()
}

func reloadExtensionRegistry() error {
	if extensions.Default == nil {
		reg, err := extensions.LoadRegistry()
		if err != nil {
			return err
		}
		extensions.Default = reg
	} else if err := extensions.Default.Reload(); err != nil {
		return err
	}
	if extensionManager != nil {
		extensionManager.SetExtensionRegistry(extensions.Default)
	}
	mentions.DefaultRegistry.SetExtensionSource(extensions.Default.MentionSource())
	return nil
}

// reloadExtensionRegistryAsync reloads hosts in the background so env-save
// responses stay fast (spawn can take seconds).
func reloadExtensionRegistryAsync() {
	go func() {
		if err := reloadExtensionRegistry(); err != nil {
			fmt.Fprintf(os.Stderr, "[extensions] reload after env save: %v\n", err)
		}
	}()
}
