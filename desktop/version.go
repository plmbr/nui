// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"os"
	"path/filepath"
	"strings"

	"nui/internal/appversion"
)

// desktopAppVersion resolves the product version for the desktop process.
// Prefer NUI_VERSION / ldflags-set appversion; fall back to bundled CLI.
func desktopAppVersion() string {
	if v := strings.TrimSpace(os.Getenv("NUI_VERSION")); v != "" {
		return strings.TrimPrefix(v, "v")
	}
	if v := appversion.Get(); v != "" && v != "dev" {
		return v
	}
	if bundled, err := bundledCLIPath(); err == nil {
		if ver, err := readCLIVersion(bundled); err == nil && ver != "" {
			return ver
		}
	}
	return appversion.Get()
}

// desktopCLIVersion returns the PATH-installed CLI version when present,
// otherwise the bundled sidecar, then the last recorded install state.
func desktopCLIVersion() string {
	if dir, err := cliInstallDir(); err == nil {
		dest := filepath.Join(dir, cliBinaryName())
		if ver, err := readCLIVersion(dest); err == nil && ver != "" {
			return ver
		}
	}
	if bundled, err := bundledCLIPath(); err == nil {
		if ver, err := readCLIVersion(bundled); err == nil && ver != "" {
			return ver
		}
	}
	if st, ok := loadCLIState(); ok {
		return strings.TrimSpace(st.Version)
	}
	return ""
}

const desktopWebsiteURL = "https://nui.plmbr.dev"

// desktopAboutMessage is the macOS About dialog informative text.
func desktopAboutMessage() string {
	return formatAboutMessage(appversion.Get(), desktopCLIVersion())
}

func formatAboutMessage(appVer, cliVer string) string {
	appVer = strings.TrimSpace(appVer)
	if appVer == "" {
		appVer = "dev"
	}
	cliVer = strings.TrimSpace(cliVer)
	if cliVer == "" {
		cliVer = "unavailable"
	}
	return "Self-hosted AI agent sessions\n\nApp version: " + appVer + "\nCLI version: " + cliVer + "\n\n" + desktopWebsiteURL
}
