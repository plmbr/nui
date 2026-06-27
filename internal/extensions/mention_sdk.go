// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const mentionSDKFile = "loop_mention.py"

// MentionSDKDir returns a directory containing loop_mention.py, installing under ~/.loop when needed.
func MentionSDKDir() (string, error) {
	if p := strings.TrimSpace(os.Getenv("LOOP_MENTION_SDK_DIR")); p != "" {
		if _, err := os.Stat(filepath.Join(p, mentionSDKFile)); err != nil {
			return "", fmt.Errorf("LOOP_MENTION_SDK_DIR %q: %w", p, err)
		}
		return p, nil
	}
	if source, err := findMentionSDKSource(); err == nil {
		return installMentionSDK(source)
	}
	return installedMentionSDKDir()
}

func installedMentionSDKDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".loop", "harness-sdk")
	if _, err := os.Stat(filepath.Join(dir, mentionSDKFile)); err != nil {
		return "", err
	}
	return dir, nil
}

func findMentionSDKSource() (string, error) {
	if p, err := findHarnessSDKFileNearExecutable(mentionSDKFile); err == nil {
		return p, nil
	}
	if p, err := findHarnessSDKFileNearWorkingDir(mentionSDKFile); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("%s not found (set LOOP_MENTION_SDK_DIR)", mentionSDKFile)
}

func findHarnessSDKFileNearExecutable(name string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "harness-sdk", name)
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("%s not found near executable", name)
}

func findHarnessSDKFileNearWorkingDir(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "harness-sdk", name)
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("%s not found near working directory", name)
}

func installMentionSDK(source string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	destDir := filepath.Join(home, ".loop", "harness-sdk")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}
	dest := filepath.Join(destDir, mentionSDKFile)
	data, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return "", err
	}
	return destDir, nil
}
