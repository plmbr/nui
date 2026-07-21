// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nui/harness-sdk"
)

const mentionSDKFile = "nui_mention.py"

// MentionSDKDir returns a directory containing nui_mention.py, installing under ~/.nui when needed.
func MentionSDKDir() (string, error) {
	if p := strings.TrimSpace(os.Getenv("NUI_MENTION_SDK_DIR")); p != "" {
		if _, err := os.Stat(filepath.Join(p, mentionSDKFile)); err != nil {
			return "", fmt.Errorf("NUI_MENTION_SDK_DIR %q: %w", p, err)
		}
		return p, nil
	}
	dir, err := harnesssdk.InstallDir()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(dir, mentionSDKFile)); err != nil {
		return "", err
	}
	return dir, nil
}
