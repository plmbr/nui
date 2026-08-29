// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"nui/internal/server"
	"nui/ui"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	defaultPort         = 8080
	maxPortScanAttempts = 100
)

// App is the Wails desktop shell around the nui HTTP server.
type App struct {
	mu         sync.RWMutex
	ctx        context.Context
	baseURL    string
	startErr   string
	ownsServer bool
	inst       *server.Instance
	port       int
	uiFS       fs.FS
	proxy      *httputil.ReverseProxy
	updater    *appUpdater
}

func NewApp() *App {
	port := defaultPort
	if v := strings.TrimSpace(os.Getenv("NUI_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			port = p
		}
	}
	return &App{port: port, uiFS: ui.DistFS()}
}

// findListenPort returns the first TCP port in [start, start+maxAttempts) that
// can be bound on host. Used to avoid attaching to an existing nui server.
func findListenPort(host string, start, maxAttempts int) (int, error) {
	for port := start; port < start+maxAttempts; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			continue
		}
		_ = ln.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no available port in range %d-%d on %s", start, start+maxAttempts-1, host)
}

// startServer starts a dedicated local nui HTTP server on the first free port
// at or above a.port (8080 by default, or NUI_PORT as scan start).
// Must run before wails.Run: on Darwin, OnStartup runs concurrently with the
// first page load, so deferring server start there races and shows a blank error.
func (a *App) startServer() error {
	const host = "127.0.0.1"
	port, err := findListenPort(host, a.port, maxPortScanAttempts)
	if err != nil {
		a.setBaseURL("", false, err)
		return err
	}

	inst, err := server.NewInstance(server.ListenConfig{
		Port:    port,
		Host:    host,
		UIFiles: a.uiFS,
		Options: server.StartOptions{},
	})
	if err != nil {
		a.setBaseURL("", false, err)
		return fmt.Errorf("configure server: %w", err)
	}
	if err := inst.StartBackground(); err != nil {
		a.setBaseURL("", false, err)
		return fmt.Errorf("start server: %w", err)
	}
	a.mu.Lock()
	a.inst = inst
	a.mu.Unlock()
	a.setBaseURL(inst.URL(), true, nil)
	fmt.Fprintf(os.Stderr, "nui desktop: server listening at %s\n", inst.URL())
	return nil
}

func (a *App) setBaseURL(urlStr string, owns bool, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.baseURL = urlStr
	a.ownsServer = owns
	a.proxy = nil
	if err != nil {
		a.startErr = err.Error()
	} else {
		a.startErr = ""
		if urlStr != "" {
			if target, parseErr := url.Parse(urlStr); parseErr == nil {
				proxy := httputil.NewSingleHostReverseProxy(target)
				// Stream SSE/AG-UI chunks as soon as they arrive.
				proxy.FlushInterval = -1
				proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
					fmt.Fprintf(os.Stderr, "nui desktop: proxy %s: %v\n", r.URL.Path, e)
					http.Error(w, "nui server unavailable", http.StatusBadGateway)
				}
				a.proxy = proxy
			}
		}
	}
}

func (a *App) getBaseURL() (string, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.baseURL, a.startErr
}

func (a *App) getProxy() *httputil.ReverseProxy {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.proxy
}

// UIAssets returns the embedded React build for the Wails asset server.
func (a *App) UIAssets() fs.FS {
	return a.uiFS
}

// onStartup stores the Wails context for runtime calls (focus on second launch).
func (a *App) onStartup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
	a.startAppUpdateChecker()
}

func (a *App) focusMainWindow() {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx == nil {
		return
	}
	runtime.WindowUnminimise(ctx)
	runtime.WindowShow(ctx)
	runtime.Show(ctx)
}

// shutdown stops the embedded server when this process owns it.
// Closes HTTP quickly and does not wait on agents/extensions — process exit
// reaps child harnesses, and Wails blocks quit until OnShutdown returns.
func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	owns := a.ownsServer
	inst := a.inst
	a.mu.Unlock()
	if !owns || inst == nil {
		return
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	if err := inst.ShutdownHTTP(stopCtx); err != nil {
		fmt.Fprintf(os.Stderr, "nui desktop: shutdown: %v\n", err)
	}
}

// assetMiddleware serves the SPA from Wails (so --wails-draggable works) and
// reverse-proxies API/SSE/MCP to the real localhost server. Keeping API on the
// wails:// origin avoids WKWebView opening extra windows for cross-origin
// http://127.0.0.1 fetches (which started after absolute apiBase URLs).
func (a *App) assetMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		baseURL, startErr := a.getBaseURL()
		if baseURL == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			msg := "nui server failed to start."
			if startErr != "" {
				msg = "nui server failed to start: " + startErr
			}
			fmt.Fprintf(w, `<!DOCTYPE html><html><body><p>%s</p></body></html>`, htmlEscape(msg))
			return
		}
		if shouldProxyToServer(r.URL.Path) {
			proxy := a.getProxy()
			if proxy == nil {
				http.Error(w, "nui server unavailable", http.StatusBadGateway)
				return
			}
			proxy.ServeHTTP(w, r)
			return
		}
		if shouldServeDesktopIndex(r.URL.Path) {
			a.serveDesktopIndex(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func shouldProxyToServer(path string) bool {
	if path == "/health" {
		return true
	}
	if strings.HasPrefix(path, "/api/") || path == "/api" {
		return true
	}
	if strings.HasPrefix(path, "/mcp-") {
		return true
	}
	return false
}

func shouldServeDesktopIndex(path string) bool {
	switch path {
	case "/", "/index.html", "/launch", "/customize", "/schedules":
		return true
	}
	return strings.HasPrefix(path, "/sessions/")
}

func (a *App) serveDesktopIndex(w http.ResponseWriter) {
	raw, err := fs.ReadFile(a.uiFS, "index.html")
	if err != nil {
		http.Error(w, "error reading index.html", http.StatusInternalServerError)
		return
	}
	// Relative /api stays on the wails:// origin and is proxied — do not set
	// __NUI_API_BASE__ to http://127.0.0.1 (that opened extra webview windows).
	inject := `<script>window.__NUI_DESKTOP__=true;</script>` +
		`<script>(function(){` +
		`function isLocal(u){try{var x=new URL(u,window.location.href);return x.hostname==='127.0.0.1'||x.hostname==='localhost';}catch(e){return false}}` +
		`var _open=window.open;` +
		`window.open=function(url){` +
		`if(!url)return null;` +
		`var s=String(url);` +
		`if(isLocal(s)||s.indexOf('wails:')===0){console.warn('nui desktop: blocked window.open',s);return null}` +
		`if(window.runtime&&typeof window.runtime.BrowserOpenURL==='function'){window.runtime.BrowserOpenURL(s);return null}` +
		`return typeof _open==='function'?_open.apply(window,arguments):null` +
		`};` +
		`document.addEventListener('click',function(e){` +
		`var a=e.target&&e.target.closest&&e.target.closest('a[href]');` +
		`if(!a)return;` +
		`var href=a.href||'';` +
		`if(isLocal(href)){e.preventDefault();e.stopPropagation();return}` +
		`if(a.target==='_blank'){e.preventDefault();e.stopPropagation();` +
		`if(window.runtime&&window.runtime.BrowserOpenURL)window.runtime.BrowserOpenURL(href)}` +
		`},true);` +
		`})();</script>`
	html := string(raw)
	if strings.Contains(html, "<head>") {
		html = strings.Replace(html, "<head>", "<head>"+inject, 1)
	} else {
		html = inject + html
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(s)
}
