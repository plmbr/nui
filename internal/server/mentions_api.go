// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"net/http"
	"strings"

	"loop/internal/mentions"
)

func handleSessionMentions(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mu.RLock()
	session, ok := findSession(sessionID)
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	query := r.URL.Query()
	parent := strings.TrimSpace(query.Get("parent"))
	limit := mentions.MaxItems
	if parent == mentions.BuiltinFilesRoot {
		limit = mentions.MaxFileItems
	}
	workingDir, err := effectiveWorkingDir(session.WorkingDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := mentions.DefaultRegistry.List(r.Context(), mentions.ListRequest{
		SessionID:  sessionID,
		WorkingDir: workingDir,
		Parent:     parent,
		Query:      strings.TrimSpace(query.Get("query")),
		Limit:      limit,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if resp.Items == nil {
		resp.Items = []mentions.Item{}
	}
	if resp.Breadcrumb == nil {
		resp.Breadcrumb = []mentions.Breadcrumb{}
	}
	writeJSON(w, http.StatusOK, struct {
		Items      []mentions.Item       `json:"items"`
		Breadcrumb []mentions.Breadcrumb `json:"breadcrumb"`
	}{
		Items:      resp.Items,
		Breadcrumb: resp.Breadcrumb,
	})
}
