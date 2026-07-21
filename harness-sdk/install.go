// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package harnesssdk

import (
	"fmt"
	"os"
	"path/filepath"
)

const installSubdir = "harness-sdk"

// InstallDir returns ~/.nui/harness-sdk, writing embedded files when missing or incomplete.
func InstallDir() (string, error) {
	return install(false)
}

// ReinstallDir force-writes all embedded harness-sdk files to ~/.nui/harness-sdk/.
func ReinstallDir() (string, error) {
	return install(true)
}

// FilePath returns the installed path for one embedded module, installing when needed.
func FilePath(name string) (string, error) {
	dir, err := InstallDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("harness-sdk %s: %w", name, err)
	}
	return path, nil
}

func install(force bool) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dest := filepath.Join(home, ".nui", installSubdir)
	if !force && installed(dest) {
		return dest, nil
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}
	for _, name := range FileNames {
		data, err := embedded.ReadFile(name)
		if err != nil {
			return "", fmt.Errorf("read embedded harness-sdk/%s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dest, name), data, 0o644); err != nil {
			return "", fmt.Errorf("write harness-sdk/%s: %w", name, err)
		}
	}
	return dest, nil
}

func installed(dest string) bool {
	for _, name := range FileNames {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			return false
		}
	}
	return true
}
