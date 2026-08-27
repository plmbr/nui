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

	"nui/harness-sdk"
	"nui/internal/agent"
	"nui/internal/extensions"
	"nui/internal/mcpoauth"
	"nui/internal/memory"
	"nui/internal/mentions"
	"nui/internal/storageext"
	"nui/internal/store"
)

var extensionManager *agent.Manager

// Instance is a configured nui HTTP server that can be started and stopped.
type Instance struct {
	srv  *http.Server
	url  string
	port int
	opts StartOptions
}

// ListenConfig configures NewInstance.
type ListenConfig struct {
	Port    int
	Host    string // empty binds all interfaces; use "127.0.0.1" for desktop
	UIFiles fs.FS
	Options StartOptions
}

// URL returns the base URL clients should use (e.g. http://127.0.0.1:8080).
func (inst *Instance) URL() string { return inst.url }

// Port returns the listening port.
func (inst *Instance) Port() int { return inst.port }

// NewInstance builds the HTTP mux and server without listening.
func NewInstance(cfg ListenConfig) (*Instance, error) {
	if cfg.UIFiles == nil {
		return nil, fmt.Errorf("UI files are required")
	}
	port := cfg.Port
	if port <= 0 {
		port = 8080
	}

	mux := http.NewServeMux()

	assetsFS, err := fs.Sub(cfg.UIFiles, "assets")
	if err != nil {
		return nil, fmt.Errorf("reading embedded assets: %w", err)
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))

	if vendorFS, err := fs.Sub(cfg.UIFiles, "vendor"); err == nil {
		mux.Handle("/vendor/", http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS))))
	}

	if err := initStore(); err != nil {
		return nil, fmt.Errorf("loading store: %w", err)
	}
	store.ApplyGlobalEnvToProcess()

	if _, err := harnesssdk.InstallDir(); err != nil {
		fmt.Fprintf(os.Stderr, "[harness-sdk] failed to install: %v\n", err)
	}

	extensionManager = agent.NewManager()

	if reg, err := extensions.LoadRegistry(); err != nil {
		fmt.Fprintf(os.Stderr, "[extensions] failed to load: %v\n", err)
		memory.SetStore(storageext.NewCoordinator(nil))
	} else {
		extensionManager.SetExtensionRegistry(reg)
		mentions.DefaultRegistry.SetExtensionSource(reg.MentionSource())
		memory.SetStore(storageext.NewCoordinator(reg))
	}

	if err := applyStartSettings(cfg.Options); err != nil {
		return nil, fmt.Errorf("apply settings: %w", err)
	}

	mcpoauth.SetListenPort(port)

	registerAPIRoutes(mux)
	registerMCPRoutes(mux)
	runScheduler()

	uiFiles := cfg.UIFiles
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

	host := strings.TrimSpace(cfg.Host)
	var addr string
	var url string
	if host == "" {
		addr = fmt.Sprintf(":%d", port)
		url = fmt.Sprintf("http://localhost:%d", port)
	} else {
		addr = fmt.Sprintf("%s:%d", host, port)
		url = fmt.Sprintf("http://%s:%d", host, port)
	}

	return &Instance{
		srv:  &http.Server{Addr: addr, Handler: withWailsCORS(mux)},
		url:  url,
		port: port,
		opts: cfg.Options,
	}, nil
}

// Serve listens until the server is shut down. It is safe to call from a goroutine.
func (inst *Instance) Serve() error {
	fmt.Printf("Listening on %s\n", inst.url)

	if needsCLILaunch(inst.opts) {
		go runCLILaunch(inst.port, inst.opts)
	} else if needsCLIOpen(inst.opts) {
		go runCLIOpen(inst.port)
	}

	if err := inst.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// StartBackground starts Serve in a goroutine and waits until /health succeeds.
func (inst *Instance) StartBackground() error {
	errCh := make(chan error, 1)
	go func() {
		if err := inst.Serve(); err != nil {
			errCh <- err
		}
	}()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			return err
		default:
		}
		client := &http.Client{Timeout: 200 * time.Millisecond}
		resp, err := client.Get(inst.url + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for server at %s", inst.url)
}

// ShutdownHTTP stops the scheduler and HTTP listener only (no agent teardown).
// Used by the desktop shell for fast window close.
func (inst *Instance) ShutdownHTTP(ctx context.Context) error {
	stopScheduler()
	if inst.srv == nil {
		return nil
	}
	return inst.srv.Shutdown(ctx)
}

// Shutdown stops the HTTP server, then agents/extensions/scheduler.
// Agent and extension teardown respects ctx so callers can bound close time
// (desktop uses ShutdownHTTP; CLI uses ~15s).
func (inst *Instance) Shutdown(ctx context.Context) error {
	fmt.Fprintln(os.Stderr, "shutting down: stopping agents and containers...")

	httpErr := inst.ShutdownHTTP(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if extensions.Default != nil {
			extensions.Default.Shutdown()
		}
		if extensionManager != nil {
			extensionManager.StopAll()
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "shutting down: timed out waiting for agents/extensions; exiting")
	}
	return httpErr
}

// Start runs the HTTP server and blocks until SIGINT/SIGTERM (CLI entrypoint).
func Start(port int, uiFiles fs.FS, opts StartOptions) error {
	inst, err := NewInstance(ListenConfig{
		Port:    port,
		UIFiles: uiFiles,
		Options: opts,
	})
	if err != nil {
		return err
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = inst.Shutdown(ctx)
	}()

	return inst.Serve()
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

// withWailsCORS allows the desktop webview (wails:// origin) to call the local API.
// Same-origin browser use is unchanged.
func withWailsCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isWailsOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID, Accept, Cache-Control")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions && isWailsOrigin(origin) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isWailsOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	if origin == "null" {
		return true
	}
	o := strings.ToLower(origin)
	return strings.HasPrefix(o, "wails:") || strings.Contains(o, "wails.localhost")
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
