// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"os"
	"strings"

	"nui/internal/pathutil"
)

// effectiveWorkingDir returns the session working directory when set, otherwise
// the nui process current working directory.
func effectiveWorkingDir(requested string) (string, error) {
	if wd := strings.TrimSpace(requested); wd != "" {
		return pathutil.ExpandHome(wd)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("working directory is required")
	}
	return wd, nil
}
