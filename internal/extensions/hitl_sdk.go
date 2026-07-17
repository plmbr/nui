// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var hitlSDKFiles = []string{
	"nui_hitl.py",
	"nui_hitl_channel.py",
	"nui_agent_stdio.py",
}

// HitlSDKDir returns a directory containing nui_hitl.py, installing harness-sdk files under ~/.nui when needed.
func HitlSDKDir() (string, error) {
	if p := strings.TrimSpace(os.Getenv("NUI_HITL_SDK_DIR")); p != "" {
		if _, err := os.Stat(filepath.Join(p, "nui_hitl.py")); err != nil {
			return "", fmt.Errorf("NUI_HITL_SDK_DIR %q: %w", p, err)
		}
		return p, nil
	}
	if srcDir, err := findHitlSDKSourceDir(); err == nil {
		return installHitlSDK(srcDir)
	}
	return installedHitlSDKDir()
}

func installedHitlSDKDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".nui", "harness-sdk")
	if _, err := os.Stat(filepath.Join(dir, "nui_hitl.py")); err != nil {
		return "", err
	}
	return dir, nil
}

func findHitlSDKSourceDir() (string, error) {
	if p, err := findHarnessSDKFileNearExecutable("nui_hitl.py"); err == nil {
		return filepath.Dir(p), nil
	}
	if p, err := findHarnessSDKFileNearWorkingDir("nui_hitl.py"); err == nil {
		return filepath.Dir(p), nil
	}
	return "", fmt.Errorf("nui_hitl.py not found (set NUI_HITL_SDK_DIR)")
}

func installHitlSDK(srcDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	destDir := filepath.Join(home, ".nui", "harness-sdk")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}
	for _, name := range hitlSDKFiles {
		data, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			return "", fmt.Errorf("read harness-sdk/%s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(destDir, name), data, 0644); err != nil {
			return "", fmt.Errorf("write harness-sdk/%s: %w", name, err)
		}
	}
	return destDir, nil
}
