// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"nui/internal/appversion"
	"nui/internal/store"
	"nui/internal/update"
)

var (
	cliUpdateMgr      = update.NewManager(update.KindCLI, "")
	updateCheckMu     sync.Mutex
	updateCheckerOnce sync.Once
)

func registerUpdateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/version", handleVersion)
	mux.HandleFunc("/api/update/status", handleUpdateStatus)
	mux.HandleFunc("/api/update/check", handleUpdateCheck)
	mux.HandleFunc("/api/update/download", handleUpdateDownload)
	mux.HandleFunc("/api/update/apply", handleUpdateApply)
	mux.HandleFunc("/api/update/skip", handleUpdateSkip)
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"version": appversion.Get(),
	})
}

func handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cliUpdateMgr.SetCurrentVersion(appversion.Get())
	st := cliUpdateMgr.Status()
	writeJSON(w, http.StatusOK, st)
}

func handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Force  bool   `json:"force"`
		Target string `json:"target"` // self | pathCli
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if !updateCheckMu.TryLock() {
		http.Error(w, "update check already in progress", http.StatusConflict)
		return
	}
	defer updateCheckMu.Unlock()

	current := appversion.Get()
	if body.Target == string(update.TargetPathCLI) {
		if v, err := pathCLIVersion(); err == nil && v != "" {
			current = v
		}
	}
	cliUpdateMgr.SetCurrentVersion(current)
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	st, err := cliUpdateMgr.Check(ctx, body.Force)
	if err != nil {
		writeJSON(w, http.StatusOK, st)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func pathCLIVersion() (string, error) {
	path, err := update.DefaultCLIPath()
	if err != nil {
		return "", err
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("path CLI not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !updateCheckMu.TryLock() {
		http.Error(w, "update operation already in progress", http.StatusConflict)
		return
	}
	defer updateCheckMu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	st, err := cliUpdateMgr.Download(ctx)
	if err != nil {
		writeJSON(w, http.StatusOK, st)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Target string `json:"target"` // self | pathCli
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	target := update.Target(body.Target)
	if target == "" {
		target = update.TargetSelf
	}
	if target != update.TargetSelf && target != update.TargetPathCLI {
		http.Error(w, `target must be "self" or "pathCli"`, http.StatusBadRequest)
		return
	}

	if !updateCheckMu.TryLock() {
		http.Error(w, "update operation already in progress", http.StatusConflict)
		return
	}
	defer updateCheckMu.Unlock()

	st, err := cliUpdateMgr.ApplyCLI(target)
	if err != nil {
		writeJSON(w, http.StatusOK, st)
		return
	}
	if target == update.TargetSelf {
		appversion.Set(st.CurrentVersion)
	}
	writeJSON(w, http.StatusOK, st)
}

func handleUpdateSkip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Version == "" {
		st := cliUpdateMgr.Status()
		body.Version = st.AvailableVersion
	}
	current, err := store.LoadUserSettings()
	if err != nil {
		http.Error(w, "failed to load settings", http.StatusInternalServerError)
		return
	}
	current.SkippedUpdateVersion = update.NormalizeVersion(body.Version)
	if err := store.SaveSettings(current); err != nil {
		http.Error(w, "failed to save settings", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"skippedUpdateVersion": current.SkippedUpdateVersion})
}

// startUpdateChecker launches a background periodic CLI update check (never auto-downloads).
func startUpdateChecker() {
	updateCheckerOnce.Do(func() {
		go runUpdateCheckerLoop()
	})
}

func runUpdateCheckerLoop() {
	// Initial delay so startup isn't blocked / raced with other init.
	time.Sleep(8 * time.Second)
	for {
		settings, err := store.LoadSettings()
		if err != nil || !store.AutoCheckUpdatesEnabled(settings) {
			time.Sleep(time.Hour)
			continue
		}
		interval := time.Duration(store.UpdateCheckInterval(settings)) * time.Hour

		if updateCheckMu.TryLock() {
			cliUpdateMgr.SetCurrentVersion(appversion.Get())
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			st, err := cliUpdateMgr.Check(ctx, false)
			cancel()
			updateCheckMu.Unlock()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[update] check failed: %v\n", err)
			} else if st.State == update.StateAvailable {
				fmt.Fprintf(os.Stderr, "[update] %s available (current %s)\n", st.AvailableVersion, st.CurrentVersion)
			}
		}
		time.Sleep(interval)
	}
}
