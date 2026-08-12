// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"context"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

func main() {
	// Finder/Dock launches strip PATH; restore login-shell + common bins so
	// builtin CLI agents (claude, pi, …) are detected and runnable.
	ensureDesktopPATH()

	// Harnesses spawn this binary as "nui-desktop viz-mcp" (etc.). Handle those
	// before the GUI / single-instance lock so MCP stdio works.
	mcpMain()

	// Bundle ships a CGO-free CLI; install to ~/.local/bin (or %LOCALAPPDATA%\nui)
	// so external MCP hosts / shells find `nui` without a separate download.
	ensureCLIInstalled()

	app := NewApp()
	// Start HTTP before the window: Darwin runs OnStartup concurrently with the
	// first page load, so deferring server start there races and shows a blank error.
	if err := app.startServer(); err != nil {
		log.Fatal(err)
	}

	err := wails.Run(&options.App{
		Title:            "nui",
		Width:            1280,
		Height:           840,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 10, A: 255},
		AssetServer: &assetserver.Options{
			Assets:     app.UIAssets(),
			Middleware: app.assetMiddleware,
		},
		OnStartup: func(ctx context.Context) {
			app.onStartup(ctx)
		},
		OnShutdown: app.shutdown,
		DragAndDrop: &options.DragAndDrop{
			// Prevent dropped files from opening as extra webview windows.
			DisableWebViewDrop: true,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "dev.plmbr.nui.desktop",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				app.focusMainWindow()
			},
		},
		Mac: &mac.Options{
			// Hidden title (no NSToolbar) — traffic lights stay overlaid.
			TitleBar: mac.TitleBarHidden(),
			About: &mac.AboutInfo{
				Title:   "nui",
				Message: "Self-hosted AI agent sessions",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
