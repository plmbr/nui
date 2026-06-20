// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loop/internal/agent"
)

var extensionManager *agent.Manager

func Start(port int, uiFiles fs.FS, opts StartOptions) error {
	mux := http.NewServeMux()

	assetsFS, err := fs.Sub(uiFiles, "assets")
	if err != nil {
		return fmt.Errorf("reading embedded assets: %w", err)
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))

	if err := initStore(); err != nil {
		return fmt.Errorf("loading store: %w", err)
	}

	extensionManager = agent.NewManager()

	if err := bootstrapFromCLI(opts); err != nil {
		return fmt.Errorf("bootstrap session: %w", err)
	}

	mu.RLock()
	entries := make([]agent.PrewarmEntry, 0, len(sessions))
	for _, s := range sessions {
		if extType := prewarmExtensionType(s.AgentType); extType != "" {
			entries = append(entries, agent.PrewarmEntry{
				SessionID:   s.ID,
				AgentType:   extType,
				WorkingDir:  s.WorkingDir,
				AgentConfig: s.AgentConfig,
			})
		}
	}
	mu.RUnlock()
	extensionManager.PrewarmSessions(entries)

	registerAPIRoutes(mux)
	registerMCPRoutes(mux)

	if err := initMCP(); err != nil {
		return fmt.Errorf("initializing MCP: %w", err)
	}

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
		fmt.Fprintln(os.Stderr, "shutting down: stopping agents and containers...")
		extensionManager.StopAll()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}
