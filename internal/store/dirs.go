// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	envDataDir       = "NUI_DATA_DIR"
	envSystemConfig  = "NUI_SYSTEM_CONFIG"
	defaultUserRel   = ".nui"
	defaultSystemDir = "/etc/nui"
)

// UserDir returns the writable per-user data directory.
// Override with NUI_DATA_DIR; otherwise ~/.nui. Creates the directory if needed.
func UserDir() (string, error) {
	if v := strings.TrimSpace(os.Getenv(envDataDir)); v != "" {
		if err := os.MkdirAll(v, 0700); err != nil {
			return "", err
		}
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, defaultUserRel)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// Dir is an alias for UserDir (writable per-user tree).
func Dir() (string, error) {
	return UserDir()
}

// SystemDir returns the read-only admin config directory.
// Override with NUI_SYSTEM_CONFIG; otherwise /etc/nui.
// Does not create the directory — missing system config is normal for desktop installs.
func SystemDir() string {
	if v := strings.TrimSpace(os.Getenv(envSystemConfig)); v != "" {
		return v
	}
	return defaultSystemDir
}

// SystemDirExists reports whether the system config directory is present.
func SystemDirExists() bool {
	info, err := os.Stat(SystemDir())
	return err == nil && info.IsDir()
}
