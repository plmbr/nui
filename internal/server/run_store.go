// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"nui/internal/agent"
	"nui/internal/store"
)

type RunStatus string

const (
	RunStatusRunning        RunStatus = "running"
	RunStatusAwaitingUser   RunStatus = "awaiting_user"
	RunStatusCompleted      RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

type RunRecord struct {
	RunID      string    `json:"runId"`
	SessionID  string    `json:"sessionId"`
	Status     RunStatus `json:"status"`
	Message    string    `json:"message,omitempty"`
	Output     string    `json:"output,omitempty"`
	Error      string    `json:"error,omitempty"`
	StartedAt  string    `json:"startedAt"`
	FinishedAt string    `json:"finishedAt,omitempty"`
}

type runLogEntry struct {
	Seq   int         `json:"seq"`
	Event agent.Event `json:"event"`
}

var (
	runStoreMu   sync.RWMutex
	runRecords   = map[string]*RunRecord{}
	sessionRuns  = map[string][]string{}
	runListeners = map[string]map[chan runLogEntry]struct{}{}
)

func createRunRecord(sessionID, runID, message string) *RunRecord {
	rec := &RunRecord{
		RunID:     runID,
		SessionID: sessionID,
		Status:    RunStatusRunning,
		Message:   message,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	runStoreMu.Lock()
	runRecords[runID] = rec
	sessionRuns[sessionID] = append(sessionRuns[sessionID], runID)
	runStoreMu.Unlock()
	return rec
}

func getRunRecord(runID string) (*RunRecord, bool) {
	runStoreMu.RLock()
	defer runStoreMu.RUnlock()
	rec, ok := runRecords[runID]
	if !ok {
		return nil, false
	}
	copy := *rec
	return &copy, true
}

func listSessionRuns(sessionID string) []RunRecord {
	runStoreMu.RLock()
	defer runStoreMu.RUnlock()
	ids := sessionRuns[sessionID]
	out := make([]RunRecord, 0, len(ids))
	for _, id := range ids {
		if rec, ok := runRecords[id]; ok {
			out = append(out, *rec)
		}
	}
	return out
}

func finishRunRecord(runID string, status RunStatus, output, errMsg string) {
	runStoreMu.Lock()
	rec, ok := runRecords[runID]
	if !ok {
		runStoreMu.Unlock()
		return
	}
	rec.Status = status
	rec.Output = output
	rec.Error = errMsg
	rec.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	sessionID := rec.SessionID
	finishedAt := rec.FinishedAt
	listeners := runListeners[runID]
	runStoreMu.Unlock()
	updateSessionLastRunAt(sessionID, finishedAt)
	notifyRunListeners(listeners)
}

func updateSessionLastRunAt(sessionID, lastRunAt string) {
	if sessionID == "" || lastRunAt == "" {
		return
	}
	mu.Lock()
	updated := false
	for i, s := range sessions {
		if s.ID == sessionID {
			sessions[i].LastRunAt = lastRunAt
			updated = true
			break
		}
	}
	var snapshot store.Data
	if updated {
		snapshot = snapshotData()
	}
	mu.Unlock()
	if updated {
		if err := store.SaveData(snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save session lastRunAt: %v\n", err)
		}
		notifySessionsChanged()
	}
}

func sessionHasRunningRun(sessionID string) bool {
	for _, rec := range listSessionRuns(sessionID) {
		if rec.Status == RunStatusRunning {
			return true
		}
	}
	return false
}

// runFinishedSentinel wakes SSE subscribers after finishRunRecord updates status.
var runFinishedSentinel = runLogEntry{Seq: -1}

func notifyRunListeners(listeners map[chan runLogEntry]struct{}) {
	for ch := range listeners {
		select {
		case ch <- runFinishedSentinel:
		default:
		}
	}
}

func appendRunEvent(runID string, seq int, ev agent.Event) error {
	entry := runLogEntry{Seq: seq, Event: ev}
	path, err := store.RunLogPath(runID)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "%s\n", data); err != nil {
		return err
	}

	runStoreMu.RLock()
	listeners := runListeners[runID]
	runStoreMu.RUnlock()
	for ch := range listeners {
		select {
		case ch <- entry:
		default:
		}
	}
	return nil
}

func readRunEvents(runID string, afterSeq int) ([]runLogEntry, error) {
	path, err := store.RunLogPath(runID)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []runLogEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var entry runLogEntry
		if err := json.Unmarshal(sc.Bytes(), &entry); err != nil {
			continue
		}
		if entry.Seq > afterSeq {
			out = append(out, entry)
		}
	}
	return out, sc.Err()
}

func subscribeRunEvents(runID string) (chan runLogEntry, func()) {
	ch := make(chan runLogEntry, 64)
	runStoreMu.Lock()
	if runListeners[runID] == nil {
		runListeners[runID] = map[chan runLogEntry]struct{}{}
	}
	runListeners[runID][ch] = struct{}{}
	runStoreMu.Unlock()
	unsub := func() {
		runStoreMu.Lock()
		delete(runListeners[runID], ch)
		if len(runListeners[runID]) == 0 {
			delete(runListeners, runID)
		}
		runStoreMu.Unlock()
		close(ch)
	}
	return ch, unsub
}

func isRunActive(runID string) bool {
	rec, ok := getRunRecord(runID)
	return ok && rec.Status == RunStatusRunning
}

// purgeSessionRuns cancels an in-flight run, drops in-memory run state, and
// deletes ~/.nui/runs/<runId>.jsonl files for the session.
func purgeSessionRuns(sessionID string) {
	if sessionID == "" {
		return
	}
	cancelActiveRun(sessionID, "")

	runStoreMu.Lock()
	ids := append([]string(nil), sessionRuns[sessionID]...)
	delete(sessionRuns, sessionID)
	var listenerSets []map[chan runLogEntry]struct{}
	for _, runID := range ids {
		delete(runRecords, runID)
		if listeners := runListeners[runID]; len(listeners) > 0 {
			listenerSets = append(listenerSets, listeners)
		}
		delete(runListeners, runID)
	}
	runStoreMu.Unlock()

	for _, listeners := range listenerSets {
		notifyRunListeners(listeners)
	}
	for _, runID := range ids {
		if err := store.RemoveRunLog(runID); err != nil {
			fmt.Fprintf(os.Stderr, "warn: remove run log %s: %v\n", runID, err)
		}
	}
}
