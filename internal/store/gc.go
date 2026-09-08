// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GCOptions controls what GarbageCollect removes.
type GCOptions struct {
	// DryRun reports what would be removed without deleting.
	DryRun bool
	// UpdateMaxAge is how old $TMPDIR/nui-update-* dirs must be before removal.
	// Zero defaults to 24h.
	UpdateMaxAge time.Duration
}

// GCResult summarizes what GarbageCollect removed (or would remove).
type GCResult struct {
	TempFiles         int   `json:"tempFiles"`
	SessionDirs       int   `json:"sessionDirs"`
	WorkspaceDirs     int   `json:"workspaceDirs"`
	RunLogs           int   `json:"runLogs"`
	UploadDirs        int   `json:"uploadDirs"`
	UpdateDirs        int   `json:"updateDirs"`
	EmptyBridgeDirs   int   `json:"emptyBridgeDirs"`
	LegacyDockerFiles int   `json:"legacyDockerFiles"`
	DataKeysPruned    int   `json:"dataKeysPruned"`
	BytesFreed        int64 `json:"bytesFreed"`
}

// LiveSessionIDs returns the set of session IDs currently recorded in data.json.
func LiveSessionIDs() (map[string]struct{}, error) {
	data, err := LoadData()
	if err != nil {
		return nil, err
	}
	live := make(map[string]struct{}, len(data.Sessions))
	for _, s := range data.Sessions {
		if s.ID != "" {
			live[s.ID] = struct{}{}
		}
	}
	return live, nil
}

// GarbageCollect removes leftover nui state: orphaned atomic-write temps, session
// config/workspace dirs not in data.json, unindexed or orphaned run logs, stale
// upload/update temp dirs, and stale keys inside data.json.
func GarbageCollect(opts GCOptions) (GCResult, error) {
	if opts.UpdateMaxAge <= 0 {
		opts.UpdateMaxAge = 24 * time.Hour
	}
	var result GCResult
	live, err := LiveSessionIDs()
	if err != nil {
		return result, err
	}
	dir, err := UserDir()
	if err != nil {
		return result, err
	}

	if err := gcTempFiles(dir, opts, &result); err != nil {
		return result, err
	}
	if err := gcSessionSubdirs(dir, "sessions", live, opts, &result.SessionDirs, &result); err != nil {
		return result, err
	}
	if err := gcSessionSubdirs(dir, "workspaces", live, opts, &result.WorkspaceDirs, &result); err != nil {
		return result, err
	}
	if err := gcRunLogs(live, opts, &result); err != nil {
		return result, err
	}
	if err := gcUploads(live, opts, &result); err != nil {
		return result, err
	}
	if err := gcUpdateDirs(opts, &result); err != nil {
		return result, err
	}
	if err := gcEmptyBridgeDir(dir, opts, &result); err != nil {
		return result, err
	}
	if err := gcLegacyDockerFiles(dir, opts, &result); err != nil {
		return result, err
	}
	if err := gcStaleDataKeys(live, opts, &result); err != nil {
		return result, err
	}
	return result, nil
}

func gcTempFiles(dir string, opts GCOptions, result *GCResult) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isNuiTempFile(name) {
			continue
		}
		path := filepath.Join(dir, name)
		removed, err := removePath(path, opts, result)
		if err != nil {
			return err
		}
		if removed {
			result.TempFiles++
		}
	}
	return nil
}

func isNuiTempFile(name string) bool {
	if !strings.HasSuffix(name, ".tmp") {
		return false
	}
	// saveJSON CreateTemp(dir, "*.tmp") and prefixed writers (secrets-*, extension-env-*, run-index-*, nui-save-*).
	return true
}

func gcSessionSubdirs(root, sub string, live map[string]struct{}, opts GCOptions, count *int, result *GCResult) error {
	base := filepath.Join(root, sub)
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if _, ok := live[id]; ok {
			continue
		}
		path := filepath.Join(base, id)
		removed, err := removePath(path, opts, result)
		if err != nil {
			return err
		}
		if removed {
			*count++
		}
	}
	return nil
}

func gcRunLogs(live map[string]struct{}, opts GCOptions, result *GCResult) error {
	type doomed struct {
		path string
		kind string // "run" | "tmp"
	}
	var toRemove []doomed

	runIndexMu.Lock()
	idx, err := loadRunIndexLocked()
	if err != nil {
		runIndexMu.Unlock()
		return err
	}
	indexed := indexedRunIDsLocked(idx)
	changed := false

	for sid, ids := range idx.BySession {
		if _, ok := live[sid]; ok {
			continue
		}
		for _, runID := range ids {
			path, pathErr := RunLogPath(runID)
			if pathErr != nil {
				runIndexMu.Unlock()
				return pathErr
			}
			toRemove = append(toRemove, doomed{path: path, kind: "run"})
			delete(indexed, runID)
		}
		delete(idx.BySession, sid)
		changed = true
	}

	runsDir, err := RunsDir()
	if err != nil {
		runIndexMu.Unlock()
		return err
	}
	entries, err := os.ReadDir(runsDir)
	if err != nil && !os.IsNotExist(err) {
		runIndexMu.Unlock()
		return err
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == runIndexFile {
			continue
		}
		name := e.Name()
		path := filepath.Join(runsDir, name)
		if strings.HasSuffix(name, ".tmp") {
			toRemove = append(toRemove, doomed{path: path, kind: "tmp"})
			continue
		}
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		runID := strings.TrimSuffix(name, ".jsonl")
		if _, ok := indexed[runID]; ok {
			continue
		}
		// Legacy / unindexed run logs cannot be mapped after restart; drop them.
		toRemove = append(toRemove, doomed{path: path, kind: "run"})
	}

	if changed && !opts.DryRun {
		if err := saveRunIndexLocked(idx); err != nil {
			runIndexMu.Unlock()
			return err
		}
	}
	runIndexMu.Unlock()

	for _, item := range toRemove {
		removed, err := removePath(item.path, opts, result)
		if err != nil {
			return err
		}
		if !removed {
			continue
		}
		switch item.kind {
		case "run":
			result.RunLogs++
		case "tmp":
			result.TempFiles++
		}
	}
	return nil
}

func gcUploads(live map[string]struct{}, opts GCOptions, result *GCResult) error {
	base := filepath.Join(os.TempDir(), "nui-uploads")
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if _, ok := live[id]; ok {
			continue
		}
		path := filepath.Join(base, id)
		removed, err := removePath(path, opts, result)
		if err != nil {
			return err
		}
		if removed {
			result.UploadDirs++
		}
	}
	return nil
}

func gcUpdateDirs(opts GCOptions, result *GCResult) error {
	tmp := os.TempDir()
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-opts.UpdateMaxAge)
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "nui-update-") {
			continue
		}
		path := filepath.Join(tmp, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		removed, err := removePath(path, opts, result)
		if err != nil {
			return err
		}
		if removed {
			result.UpdateDirs++
		}
	}
	return nil
}

func gcEmptyBridgeDir(root string, opts GCOptions, result *GCResult) error {
	path := filepath.Join(root, "vscode-bridge-instances")
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return nil
	}
	removed, err := removePath(path, opts, result)
	if err != nil {
		return err
	}
	if removed {
		result.EmptyBridgeDirs++
	}
	return nil
}

// legacyDockerConfigFiles are former global docker snapshot/override paths under ~/.nui.
// New launches stage these under sessions/<id>/ or a temp dir instead.
var legacyDockerConfigFiles = []string{
	".claude-snapshot.json",
	".claude-settings-override.json",
	".codex-snapshot.json",
	".codex-settings-override.json",
	".pi-settings-override.json",
	".pi-snapshot.json",
}

func gcLegacyDockerFiles(root string, opts GCOptions, result *GCResult) error {
	for _, name := range legacyDockerConfigFiles {
		path := filepath.Join(root, name)
		removed, err := removePath(path, opts, result)
		if err != nil {
			return err
		}
		if removed {
			result.LegacyDockerFiles++
		}
	}
	return nil
}

func gcStaleDataKeys(live map[string]struct{}, opts GCOptions, result *GCResult) error {
	data, err := LoadData()
	if err != nil {
		return err
	}
	pruned := 0
	for id := range data.AgentSessions {
		if _, ok := live[id]; ok {
			continue
		}
		delete(data.AgentSessions, id)
		pruned++
	}
	for id := range data.SessionMessages {
		if _, ok := live[id]; ok {
			continue
		}
		delete(data.SessionMessages, id)
		pruned++
	}
	if pruned == 0 {
		return nil
	}
	result.DataKeysPruned = pruned
	if opts.DryRun {
		return nil
	}
	if err := SaveData(data); err != nil {
		fmt.Fprintf(os.Stderr, "[gc] skip data.json key prune: %v\n", err)
		result.DataKeysPruned = 0
		return nil
	}
	return nil
}

func removePath(path string, opts GCOptions, result *GCResult) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	size := dirSize(path, info)
	if opts.DryRun {
		result.BytesFreed += size
		return true, nil
	}
	if err := os.RemoveAll(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// Skip locked / permission-denied paths so one bad file does not abort GC.
		if os.IsPermission(err) || isNotPermitted(err) {
			fmt.Fprintf(os.Stderr, "[gc] skip %s: %v\n", path, err)
			return false, nil
		}
		return false, err
	}
	result.BytesFreed += size
	return true, nil
}

func isNotPermitted(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "operation not permitted")
}

func dirSize(path string, info fs.FileInfo) int64 {
	if !info.IsDir() {
		return info.Size()
	}
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		total += fi.Size()
		return nil
	})
	return total
}

// FormatGCResult returns a one-line human summary.
func FormatGCResult(r GCResult) string {
	return fmt.Sprintf(
		"temp=%d sessions=%d workspaces=%d runs=%d uploads=%d updates=%d bridge=%d legacyDocker=%d dataKeys=%d freed=%s",
		r.TempFiles, r.SessionDirs, r.WorkspaceDirs, r.RunLogs, r.UploadDirs, r.UpdateDirs, r.EmptyBridgeDirs, r.LegacyDockerFiles, r.DataKeysPruned,
		formatBytes(r.BytesFreed),
	)
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// MarshalGCResultJSON is used by tests / CLI --json.
func MarshalGCResultJSON(r GCResult) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
