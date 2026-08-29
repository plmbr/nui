// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"nui/internal/appversion"
	"nui/internal/store"
	"nui/internal/update"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Desktop self-update track (Electron-style): check → notify → confirm download → confirm install/relaunch.
type appUpdater struct {
	mu  sync.Mutex
	mgr *update.Manager
}

func (a *App) appUpdate() *update.Manager {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.updater == nil {
		a.updater = &appUpdater{mgr: update.NewManager(update.KindDesktop, appversion.Get())}
	}
	return a.updater.mgr
}

// CheckForAppUpdate checks GitHub for a newer nui-desktop release.
func (a *App) CheckForAppUpdate(force bool) update.Status {
	mgr := a.appUpdate()
	mgr.SetCurrentVersion(appversion.Get())
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	st, err := mgr.Check(ctx, force)
	if err != nil {
		return st
	}
	a.emitUpdateEvent("update:app:status", st)
	if st.State == update.StateAvailable {
		a.emitUpdateEvent("update:app:available", st)
	}
	return st
}

// DownloadAppUpdate downloads the previously checked desktop archive.
func (a *App) DownloadAppUpdate() update.Status {
	mgr := a.appUpdate()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	st, err := mgr.Download(ctx)
	a.emitUpdateEvent("update:app:status", st)
	if err == nil && st.State == update.StateReady {
		a.emitUpdateEvent("update:app:ready", st)
	}
	return st
}

// AppUpdateStatus returns the current desktop updater status.
func (a *App) AppUpdateStatus() update.Status {
	return a.appUpdate().Status()
}

// QuitAndInstall applies the downloaded desktop update and relaunches.
func (a *App) QuitAndInstall() update.Status {
	mgr := a.appUpdate()
	st, exe, err := mgr.ApplyDesktop()
	a.emitUpdateEvent("update:app:status", st)
	if err != nil {
		return st
	}
	if exe != "" {
		if err := update.RelaunchDesktop(exe); err != nil {
			fmt.Fprintf(os.Stderr, "nui desktop: relaunch: %v\n", err)
		}
	}
	// Give the new process a moment, then quit.
	go func() {
		time.Sleep(400 * time.Millisecond)
		a.mu.RLock()
		ctx := a.ctx
		a.mu.RUnlock()
		if ctx != nil {
			runtime.Quit(ctx)
		} else {
			os.Exit(0)
		}
	}()
	return st
}

func (a *App) emitUpdateEvent(name string, st update.Status) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx == nil {
		return
	}
	runtime.EventsEmit(ctx, name, st)
}

// startAppUpdateChecker runs Electron-style periodic checks (notify only).
func (a *App) startAppUpdateChecker() {
	go func() {
		time.Sleep(10 * time.Second)
		for {
			settings, err := store.LoadSettings()
			if err != nil || !store.AutoCheckUpdatesEnabled(settings) {
				time.Sleep(time.Hour)
				continue
			}
			st := a.CheckForAppUpdate(false)
			if st.State == update.StateAvailable {
				fmt.Fprintf(os.Stderr, "nui desktop: app update available: %s\n", st.AvailableVersion)
			}
			interval := time.Duration(store.UpdateCheckInterval(settings)) * time.Hour
			time.Sleep(interval)
		}
	}()
}
