// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"loop/internal/model"
)

type activeRun struct {
	sessionID string
	runID     string
	cancel    context.CancelFunc
}

var (
	activeRunsMu        sync.Mutex
	activeRunsBySession = map[string]*activeRun{}
	activeRunsByID      = map[string]*activeRun{}
)

func runStatusIsActive(status RunStatus) bool {
	return status == RunStatusRunning || status == RunStatusAwaitingUser
}

func registerActiveRun(sessionID, runID string, cancel context.CancelFunc) {
	activeRunsMu.Lock()
	defer activeRunsMu.Unlock()
	// Runs are serialized per session in handleSessionAGUI; do not cancel the prior
	// run here or a fast follow-up message can drop the in-flight Claude turn.
	ar := &activeRun{sessionID: sessionID, runID: runID, cancel: cancel}
	activeRunsBySession[sessionID] = ar
	if runID != "" {
		activeRunsByID[runID] = ar
	}
}

func unregisterActiveRun(sessionID, runID string) {
	activeRunsMu.Lock()
	defer activeRunsMu.Unlock()
	if ar, ok := activeRunsBySession[sessionID]; ok && (runID == "" || ar.runID == runID) {
		delete(activeRunsBySession, sessionID)
		if ar.runID != "" {
			delete(activeRunsByID, ar.runID)
		}
	} else if runID != "" {
		if ar, ok := activeRunsByID[runID]; ok {
			delete(activeRunsByID, runID)
			delete(activeRunsBySession, ar.sessionID)
		}
	}
}

func cancelActiveRun(sessionID, runID string) bool {
	activeRunsMu.Lock()
	var ar *activeRun
	if runID != "" {
		ar = activeRunsByID[runID]
	} else {
		ar = activeRunsBySession[sessionID]
	}
	if ar != nil {
		delete(activeRunsBySession, ar.sessionID)
		if ar.runID != "" {
			delete(activeRunsByID, ar.runID)
		}
	}
	activeRunsMu.Unlock()
	if ar != nil {
		ar.cancel()
		return true
	}
	return false
}

func handleSessionStop(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mu.RLock()
	_, ok := findSession(sessionID)
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	runID := strings.TrimSpace(r.URL.Query().Get("runId"))
	cancelActiveRun(sessionID, runID)
	if extensionManager != nil {
		extensionManager.Stop(sessionID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func handleSessionRuns(w http.ResponseWriter, r *http.Request, sessionID string) {
	mu.RLock()
	_, ok := findSession(sessionID)
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, listSessionRuns(sessionID))
	case http.MethodPost:
		handleStartSessionRun(w, r, sessionID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleStartSessionRun(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req struct {
		Message string `json:"message"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	mu.RLock()
	session, ok := findSession(sessionID)
	agentSessionID := agentSessions[sessionID]
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		if def, found := findADLDef(session.AgentType); found && model.IsADLAutoPrompt(def) {
			message = model.ResolveADLLaunchPrompt(def, "")
		}
	}
	if message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	runID := uuid.NewString()
	createRunRecord(sessionID, runID, message)
	appendUserMessage(sessionID, message)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"runId":     runID,
		"sessionId": sessionID,
		"status":    RunStatusRunning,
	})

	go runInBackground(sessionID, session, agentSessionID, runID, message, false)
}

func runInBackground(sessionID string, session model.Session, agentSessionID, runID, message string, resolveMentions bool) {
	runCtx, cancel := context.WithCancel(context.Background())
	registerActiveRun(sessionID, runID, cancel)
	defer unregisterActiveRun(sessionID, runID)

	result := executeRun(runCtx, executeRunOptions{
		SessionID:       sessionID,
		Session:         session,
		Message:         message,
		ResolveMentions: resolveMentions,
		RunID:           runID,
		PersistLog:      true,
		AgentSessionID:  agentSessionID,
	})

	var status RunStatus
	var errMsg string
	switch {
	case result.Cancelled:
		status = RunStatusCancelled
		errMsg = "cancelled"
	case result.Errored:
		status = RunStatusFailed
		errMsg = result.ErrorMessage
	default:
		status = RunStatusCompleted
	}

	persistAssistantTurn(sessionID, result.AssistantContent, result.NewAgentSessionID, isSessionMultiStepWorkflow(session))
	finishRunRecord(runID, status, result.AssistantContent, errMsg)
}

func isSessionMultiStepWorkflow(session model.Session) bool {
	def, ok := findADLDef(session.AgentType)
	return ok && model.IsMultiStepWorkflow(def)
}

func handleSessionRunsRouter(w http.ResponseWriter, r *http.Request, sessionID, rest string) {
	rest = strings.TrimPrefix(rest, "runs")
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		handleSessionRuns(w, r, sessionID)
		return
	}
	runID, sub, _ := strings.Cut(rest, "/")
	if sub == "events" {
		handleRunEvents(w, r, sessionID, runID)
		return
	}
	if sub == "hitl" && r.Method == http.MethodPost {
		handleSessionHITLCreate(w, r, sessionID, runID)
		return
	}
	if sub != "" {
		http.NotFound(w, r)
		return
	}
	handleSessionRunGet(w, r, sessionID, runID)
}

func handleSessionRunGet(w http.ResponseWriter, r *http.Request, sessionID, runID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mu.RLock()
	_, ok := findSession(sessionID)
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	rec, ok := getRunRecord(runID)
	if !ok || rec.SessionID != sessionID {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, rec)
}

func handleRunEvents(w http.ResponseWriter, r *http.Request, sessionID, runID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rec, ok := getRunRecord(runID)
	if !ok || rec.SessionID != sessionID {
		http.NotFound(w, r)
		return
	}

	afterSeq := 0
	if lastID := strings.TrimSpace(r.Header.Get("Last-Event-ID")); lastID != "" {
		if n, err := strconv.Atoi(lastID); err == nil {
			afterSeq = n
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	past, err := readRunEvents(runID, afterSeq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lastSeq := afterSeq
	for _, entry := range past {
		writeRunSSE(w, flusher, entry)
		lastSeq = entry.Seq
	}

	if !runStatusIsActive(rec.Status) {
		writeRunSSEDone(w, flusher, rec)
		return
	}

	ch, unsub := subscribeRunEvents(runID)
	defer unsub()

	if updated, ok := getRunRecord(runID); ok && !runStatusIsActive(updated.Status) {
		writeRunSSEDone(w, flusher, updated)
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case entry, open := <-ch:
			if !open {
				if updated, ok := getRunRecord(runID); ok {
					writeRunSSEDone(w, flusher, updated)
				}
				return
			}
			if entry.Seq == runFinishedSentinel.Seq {
				if updated, ok := getRunRecord(runID); ok {
					writeRunSSEDone(w, flusher, updated)
				}
				return
			}
			if entry.Seq <= lastSeq {
				continue
			}
			writeRunSSE(w, flusher, entry)
			lastSeq = entry.Seq
			if updated, ok := getRunRecord(runID); ok && !runStatusIsActive(updated.Status) {
				writeRunSSEDone(w, flusher, updated)
				return
			}
		}
	}
}

func writeRunSSE(w http.ResponseWriter, flusher http.Flusher, entry runLogEntry) {
	data, _ := json.Marshal(entry.Event)
	fmt.Fprintf(w, "id: %d\n", entry.Seq)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func writeRunSSEDone(w http.ResponseWriter, flusher http.Flusher, rec *RunRecord) {
	payload, _ := json.Marshal(map[string]any{
		"type":   "run_finished",
		"status": rec.Status,
		"output": rec.Output,
		"error":  rec.Error,
	})
	fmt.Fprintf(w, "data: %s\n\n", payload)
	flusher.Flush()
}
