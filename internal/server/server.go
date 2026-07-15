// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"loop/internal/agent"
	"loop/internal/extensions"
	"loop/internal/mentions"
)

var extensionManager *agent.Manager

func Start(port int, uiFiles fs.FS, opts StartOptions) error {
	mux := http.NewServeMux()

	assetsFS, err := fs.Sub(uiFiles, "assets")
	if err != nil {
		return fmt.Errorf("reading embedded assets: %w", err)
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))

	if vendorFS, err := fs.Sub(uiFiles, "vendor"); err == nil {
		mux.Handle("/vendor/", http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS))))
	}

	if err := initStore(); err != nil {
		return fmt.Errorf("loading store: %w", err)
	}

	extensionManager = agent.NewManager()

	if reg, err := extensions.LoadRegistry(); err != nil {
		fmt.Fprintf(os.Stderr, "[extensions] failed to load: %v\n", err)
	} else {
		extensionManager.SetExtensionRegistry(reg)
		mentions.DefaultRegistry.SetExtensionSource(reg.MentionSource())
	}

	if err := applyStartSettings(opts); err != nil {
		return fmt.Errorf("apply settings: %w", err)
	}

	registerAPIRoutes(mux)
	registerMCPRoutes(mux)
	runScheduler()

	mux.HandleFunc("/health", handleHealth)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !isUIRoute(r.URL.Path) {
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
	url := fmt.Sprintf("http://localhost%s", addr)
	fmt.Printf("Listening on %s\n", url)

	srv := &http.Server{Addr: addr, Handler: mux}

	if needsCLILaunch(opts) {
		go runCLILaunch(port, opts)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Fprintln(os.Stderr, "shutting down: stopping agents and containers...")
		stopScheduler()
		if extensions.Default != nil {
			extensions.Default.Shutdown()
		}
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

func isUIRoute(path string) bool {
	switch path {
	case "/", "/launch", "/customize", "/schedules":
		return true
	}
	if strings.HasPrefix(path, "/sessions/") {
		return true
	}
	return false
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

func waitForHealth(baseURL string) {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	healthURL := baseURL + "/health"
	for range 50 {
		resp, err := client.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}
