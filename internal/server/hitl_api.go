// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"nui/internal/hitl"
)

func registerHITLRoutes(mux *http.ServeMux) {
	initHITL()
	mux.HandleFunc("/api/hitl/requests", handleHITLRequests)
	mux.HandleFunc("/api/hitl/requests/", handleHITLRequestByID)
	mux.HandleFunc("/api/hitl-channels", handleHITLChannels)
}

func handleHITLRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleHITLCreate(w, r)
	case http.MethodGet:
		handleHITLList(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleHITLRequestByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/hitl/requests/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	requestID := parts[0]
	if len(parts) == 2 && parts[1] == "wait" && r.Method == http.MethodGet {
		handleHITLWait(w, r, requestID)
		return
	}
	if len(parts) == 2 && parts[1] == "respond" && r.Method == http.MethodPost {
		handleHITLRespond(w, r, requestID)
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleHITLGet(w, r, requestID)
	case http.MethodDelete:
		handleHITLCancel(w, r, requestID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleHITLCreate(w http.ResponseWriter, r *http.Request) {
	var body hitl.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	createHITLRequest(w, r, body)
}

func createHITLRequest(w http.ResponseWriter, r *http.Request, body hitl.CreateInput) {
	mu.RLock()
	session, hasSession := findSession(body.SessionID)
	mu.RUnlock()
	if hasSession {
		if def, ok := findADLDef(session.AgentType); ok {
			body.Routing = defaultHITLChannels(body, def)
		}
	}

	req, err := coordinator().Create(r.Context(), body)
	if err != nil {
		if errors.Is(err, hitl.ErrHitlDisabled) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, hitl.ErrDuplicateRequest) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if body.RunID != "" {
		setRunAwaitingUser(body.RunID, true)
	}
	writeJSON(w, http.StatusCreated, req)
}

func handleHITLGet(w http.ResponseWriter, r *http.Request, requestID string) {
	req, err := coordinator().Get(r.Context(), requestID)
	if err != nil {
		if errors.Is(err, hitl.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func handleHITLList(w http.ResponseWriter, r *http.Request) {
	filter := hitl.ListFilter{
		SessionID: r.URL.Query().Get("sessionId"),
		RunID:     r.URL.Query().Get("runId"),
		Status:    r.URL.Query().Get("status"),
	}
	if r.URL.Query().Get("status") == "pending" || r.URL.Query().Get("pending") == "true" {
		filter.PendingOnly = true
	}
	list, err := coordinator().ListPending(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func handleHITLWait(w http.ResponseWriter, r *http.Request, requestID string) {
	resp, err := coordinator().Wait(r.Context(), requestID)
	if err != nil {
		if errors.Is(err, hitl.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if r.Context().Err() != nil {
			http.Error(w, "request cancelled", http.StatusRequestTimeout)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req, err := coordinator().Get(r.Context(), requestID); err == nil && req.RunID != "" {
		setRunAwaitingUser(req.RunID, false)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleHITLRespond(w http.ResponseWriter, r *http.Request, requestID string) {
	var body hitl.RespondInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.RespondedBy.Channel == "" {
		body.RespondedBy.Channel = hitl.ChannelnuiUI
	}
	resp, err := coordinator().Respond(r.Context(), requestID, body)
	if err != nil {
		if errors.Is(err, hitl.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, hitl.ErrAlreadyResolved) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req, err := coordinator().Get(r.Context(), requestID); err == nil && req.RunID != "" {
		setRunAwaitingUser(req.RunID, false)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleHITLCancel(w http.ResponseWriter, r *http.Request, requestID string) {
	if err := coordinator().Cancel(r.Context(), requestID, "cancelled"); err != nil {
		if errors.Is(err, hitl.ErrNotFound) {
		 http.NotFound(w, r)
		 return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleSessionHITLCreate(w http.ResponseWriter, r *http.Request, sessionID, runID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body hitl.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && r.ContentLength > 0 {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	body.SessionID = sessionID
	body.RunID = runID
	createHITLRequest(w, r, body)
}

func handleHITLChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channels := listHITLChannelRefs()
	writeJSON(w, http.StatusOK, channels)
}

func setRunAwaitingUser(runID string, awaiting bool) {
	runStoreMu.Lock()
	defer runStoreMu.Unlock()
	rec, ok := runRecords[runID]
	if !ok {
		return
	}
	if awaiting {
		rec.Status = RunStatusAwaitingUser
	} else if rec.Status == RunStatusAwaitingUser {
		rec.Status = RunStatusRunning
	}
}
