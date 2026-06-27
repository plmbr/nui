// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"net/http"

	"loop/internal/extensions"
	"loop/internal/mentions"
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
	if extensions.Default == nil {
		reg, err := extensions.LoadRegistry()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		extensions.Default = reg
	} else if err := extensions.Default.Reload(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if extensionManager != nil {
		extensionManager.SetExtensionRegistry(extensions.Default)
	}
	mentions.DefaultRegistry.SetExtensionSource(extensions.Default.MentionSource())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
