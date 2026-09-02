// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	envDataDir         = "NUI_DATA_DIR"
	envSystemConfig    = "NUI_SYSTEM_CONFIG"
	envExtraConfigDirs = "NUI_EXTRA_CONFIG_DIRS"
	defaultUserRel     = ".nui"
	defaultSystemDir   = "/etc/nui"
)

// ConfigRoot is a config directory root and whether nui may write under it.
type ConfigRoot struct {
	Path     string
	Writable bool
}

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

// ExtraConfigDirs returns supplemental read-only config roots from NUI_EXTRA_CONFIG_DIRS.
func ExtraConfigDirs() []string {
	raw := strings.TrimSpace(os.Getenv(envExtraConfigDirs))
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range filepath.SplitList(raw) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

// AppendExtraConfigDir appends a path to NUI_EXTRA_CONFIG_DIRS for this process.
func AppendExtraConfigDir(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	current := strings.Join(ExtraConfigDirs(), string(os.PathListSeparator))
	if current == "" {
		_ = os.Setenv(envExtraConfigDirs, dir)
		return
	}
	_ = os.Setenv(envExtraConfigDirs, current+string(os.PathListSeparator)+dir)
}

// ConfigRoots returns ordered config roots: extra dirs, system (if present), user.
func ConfigRoots() ([]ConfigRoot, error) {
	var roots []ConfigRoot
	for _, d := range ExtraConfigDirs() {
		roots = append(roots, ConfigRoot{Path: d, Writable: false})
	}
	if SystemDirExists() {
		roots = append(roots, ConfigRoot{Path: SystemDir(), Writable: false})
	}
	user, err := UserDir()
	if err != nil {
		return nil, err
	}
	roots = append(roots, ConfigRoot{Path: user, Writable: true})
	return roots, nil
}

func agentsDirIfExists(root string) string {
	dir := filepath.Join(root, "agents")
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ""
	}
	return dir
}

// AgentConfigDirs returns ordered agent definition directories (later wins on same id).
func AgentConfigDirs() ([]string, error) {
	roots, err := ConfigRoots()
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, root := range roots {
		if d := agentsDirIfExists(root.Path); d != "" {
			dirs = append(dirs, d)
		}
	}
	userAgents, err := AgentsDir()
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 || dirs[len(dirs)-1] != userAgents {
		dirs = append(dirs, userAgents)
	}
	return dirs, nil
}

func extensionsDirIfExists(root string) string {
	dir := filepath.Join(root, "extensions")
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ""
	}
	return dir
}

// ExtensionConfigDirs returns ordered extension scan directories (later wins on same name).
func ExtensionConfigDirs() ([]string, error) {
	roots, err := ConfigRoots()
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, root := range roots {
		if d := extensionsDirIfExists(root.Path); d != "" {
			dirs = append(dirs, d)
		}
	}
	userExt, err := ExtensionsDir()
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 || dirs[len(dirs)-1] != userExt {
		dirs = append(dirs, userExt)
	}
	return dirs, nil
}
