// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var hitlSDKFiles = []string{
	"loop_hitl.py",
	"loop_hitl_channel.py",
	"loop_agent_stdio.py",
}

// HitlSDKDir returns a directory containing loop_hitl.py, installing harness-sdk files under ~/.loop when needed.
func HitlSDKDir() (string, error) {
	if p := strings.TrimSpace(os.Getenv("LOOP_HITL_SDK_DIR")); p != "" {
		if _, err := os.Stat(filepath.Join(p, "loop_hitl.py")); err != nil {
			return "", fmt.Errorf("LOOP_HITL_SDK_DIR %q: %w", p, err)
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
	dir := filepath.Join(home, ".loop", "harness-sdk")
	if _, err := os.Stat(filepath.Join(dir, "loop_hitl.py")); err != nil {
		return "", err
	}
	return dir, nil
}

func findHitlSDKSourceDir() (string, error) {
	if p, err := findHarnessSDKFileNearExecutable("loop_hitl.py"); err == nil {
		return filepath.Dir(p), nil
	}
	if p, err := findHarnessSDKFileNearWorkingDir("loop_hitl.py"); err == nil {
		return filepath.Dir(p), nil
	}
	return "", fmt.Errorf("loop_hitl.py not found (set LOOP_HITL_SDK_DIR)")
}

func installHitlSDK(srcDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	destDir := filepath.Join(home, ".loop", "harness-sdk")
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
