// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

const runIndexFile = "index.json"

var runIndexMu sync.Mutex

type runIndex struct {
	BySession map[string][]string `json:"bySession"`
}

func runIndexPath() (string, error) {
	dir, err := RunsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, runIndexFile), nil
}

func loadRunIndexLocked() (runIndex, error) {
	path, err := runIndexPath()
	if err != nil {
		return runIndex{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return runIndex{BySession: map[string][]string{}}, nil
	}
	if err != nil {
		return runIndex{}, err
	}
	var idx runIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return runIndex{BySession: map[string][]string{}}, nil
	}
	if idx.BySession == nil {
		idx.BySession = map[string][]string{}
	}
	return idx, nil
}

func saveRunIndexLocked(idx runIndex) error {
	path, err := runIndexPath()
	if err != nil {
		return err
	}
	if idx.BySession == nil {
		idx.BySession = map[string][]string{}
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "run-index-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil {
		os.Remove(tmpPath)
		return werr
	}
	if cerr != nil {
		os.Remove(tmpPath)
		return cerr
	}
	return os.Rename(tmpPath, path)
}

// RegisterSessionRun records that runID belongs to sessionID for durable cleanup.
func RegisterSessionRun(sessionID, runID string) error {
	if sessionID == "" || runID == "" {
		return nil
	}
	runIndexMu.Lock()
	defer runIndexMu.Unlock()
	idx, err := loadRunIndexLocked()
	if err != nil {
		return err
	}
	list := idx.BySession[sessionID]
	for _, id := range list {
		if id == runID {
			return nil
		}
	}
	idx.BySession[sessionID] = append(list, runID)
	return saveRunIndexLocked(idx)
}

// SessionRunIDs returns run IDs previously registered for sessionID.
func SessionRunIDs(sessionID string) ([]string, error) {
	if sessionID == "" {
		return nil, nil
	}
	runIndexMu.Lock()
	defer runIndexMu.Unlock()
	idx, err := loadRunIndexLocked()
	if err != nil {
		return nil, err
	}
	list := idx.BySession[sessionID]
	out := make([]string, len(list))
	copy(out, list)
	return out, nil
}

// RemoveSessionRunLogs deletes run jsonl files for sessionID and drops the index entry.
func RemoveSessionRunLogs(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	runIndexMu.Lock()
	defer runIndexMu.Unlock()
	idx, err := loadRunIndexLocked()
	if err != nil {
		return err
	}
	ids := idx.BySession[sessionID]
	delete(idx.BySession, sessionID)
	var firstErr error
	for _, runID := range ids {
		if err := removeRunLogUnlocked(runID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := saveRunIndexLocked(idx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// AllIndexedSessionIDs returns session IDs present in the run index.
func AllIndexedSessionIDs() ([]string, error) {
	runIndexMu.Lock()
	defer runIndexMu.Unlock()
	idx, err := loadRunIndexLocked()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(idx.BySession))
	for sid := range idx.BySession {
		out = append(out, sid)
	}
	return out, nil
}

// indexedRunIDsLocked returns the set of all run IDs in the index.
func indexedRunIDsLocked(idx runIndex) map[string]struct{} {
	out := map[string]struct{}{}
	for _, ids := range idx.BySession {
		for _, id := range ids {
			out[id] = struct{}{}
		}
	}
	return out
}
