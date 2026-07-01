// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
)

type activeRun struct {
	cancel context.CancelFunc
}

var (
	activeRunsMu sync.Mutex
	activeRuns   = map[string]*activeRun{}
)

func registerActiveRun(sessionID string, cancel context.CancelFunc) {
	activeRunsMu.Lock()
	defer activeRunsMu.Unlock()
	if prev, ok := activeRuns[sessionID]; ok {
		prev.cancel()
	}
	activeRuns[sessionID] = &activeRun{cancel: cancel}
}

func unregisterActiveRun(sessionID string) {
	activeRunsMu.Lock()
	delete(activeRuns, sessionID)
	activeRunsMu.Unlock()
}

func cancelActiveRun(sessionID string) bool {
	activeRunsMu.Lock()
	run, ok := activeRuns[sessionID]
	if ok {
		delete(activeRuns, sessionID)
	}
	activeRunsMu.Unlock()
	if ok {
		run.cancel()
	}
	return ok
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

	cancelActiveRun(sessionID)
	if extensionManager != nil {
		extensionManager.Stop(sessionID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
