// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandHome replaces a leading ~ or ~/ with the user's home directory.
func ExpandHome(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return home, nil
	}
	sep := string(filepath.Separator)
	if strings.HasPrefix(path, "~"+sep) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
