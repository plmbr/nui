// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"os"
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
