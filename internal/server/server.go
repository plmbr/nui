// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"loop/internal/agent"
)

var extensionManager *agent.Manager

func Start(port int, uiFiles fs.FS, extFiles embed.FS) error {
	mux := http.NewServeMux()

	assetsFS, err := fs.Sub(uiFiles, "assets")
	if err != nil {
		return fmt.Errorf("reading embedded assets: %w", err)
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))

	if err := initStore(); err != nil {
		return fmt.Errorf("loading store: %w", err)
	}

	extDir, err := extractExtensions(extFiles)
	if err != nil {
		return fmt.Errorf("extracting extensions: %w", err)
	}
	extensionManager = agent.NewManager(extDir)
	mu.RLock()
	entries := make([]agent.PrewarmEntry, 0, len(projects))
	for _, p := range projects {
		entries = append(entries, agent.PrewarmEntry{ProjectID: p.ID, AgentType: p.AgentType, WorkingDir: p.WorkingDir, AgentConfig: p.AgentConfig})
	}
	mu.RUnlock()
	extensionManager.PrewarmProjects(entries)

	registerAPIRoutes(mux)

	mux.HandleFunc("/health", handleHealth)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		file, err := fs.ReadFile(uiFiles, "index.html")
		if err != nil {
			http.Error(w, "error reading index.html", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(file)
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Listening on http://localhost%s\n", addr)

	srv := &http.Server{Addr: addr, Handler: mux}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Fprintln(os.Stderr, "shutting down: terminating extension processes...")
		extensionManager.StopAll()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func extractExtensions(extFiles embed.FS) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	dest := filepath.Join(home, ".loop", "bin", "extensions")
	if err := os.MkdirAll(dest, 0755); err != nil {
		return "", fmt.Errorf("create extensions dir: %w", err)
	}
	entries, err := extFiles.ReadDir("extensions")
	if err != nil {
		return "", fmt.Errorf("read embedded extensions: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := extFiles.ReadFile("extensions/" + e.Name())
		if err != nil {
			return "", fmt.Errorf("read extension %s: %w", e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dest, e.Name()), data, 0755); err != nil {
			return "", fmt.Errorf("write extension %s: %w", e.Name(), err)
		}
	}
	return dest, nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}
