// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nui/harness-sdk"
)

// HitlSDKDir returns a directory containing nui_hitl.py, installing harness-sdk files under ~/.nui when needed.
func HitlSDKDir() (string, error) {
	if p := strings.TrimSpace(os.Getenv("NUI_HITL_SDK_DIR")); p != "" {
		if _, err := os.Stat(filepath.Join(p, "nui_hitl.py")); err != nil {
			return "", fmt.Errorf("NUI_HITL_SDK_DIR %q: %w", p, err)
		}
		return p, nil
	}
	dir, err := harnesssdk.InstallDir()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(dir, "nui_hitl.py")); err != nil {
		return "", err
	}
	return dir, nil
}
