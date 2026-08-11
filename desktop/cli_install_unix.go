// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ensureCLIInstallDirOnPATH(dir string) error {
	dir = filepath.Clean(dir)
	if pathListContains(os.Getenv("PATH"), dir) {
		return nil
	}
	// Current process (and children this session).
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	profile, err := loginProfilePath()
	if err != nil {
		return err
	}
	return appendPATHBlock(profile, dir)
}

func loginProfilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	shell := filepath.Base(strings.TrimSpace(os.Getenv("SHELL")))
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zprofile"), nil
	case "bash":
		bashProfile := filepath.Join(home, ".bash_profile")
		if st, err := os.Stat(bashProfile); err == nil && !st.IsDir() {
			return bashProfile, nil
		}
		return filepath.Join(home, ".profile"), nil
	default:
		profile := filepath.Join(home, ".profile")
		if st, err := os.Stat(profile); err == nil && !st.IsDir() {
			return profile, nil
		}
		return filepath.Join(home, ".zprofile"), nil
	}
}

func appendPATHBlock(profile, dir string) error {
	existing, err := os.ReadFile(profile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(existing)
	if strings.Contains(content, cliPathBlockBegin) {
		return nil
	}
	block := fmt.Sprintf("\n%s\nexport PATH=%q:${PATH}\n%s\n", cliPathBlockBegin, dir, cliPathBlockEnd)
	f, err := os.OpenFile(profile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(block)
	return err
}
