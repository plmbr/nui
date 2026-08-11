// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func ensureCLIInstallDirOnPATH(dir string) error {
	dir = filepath.Clean(dir)
	if pathListContains(os.Getenv("PATH"), dir) {
		return nil
	}
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open user Environment: %w", err)
	}
	defer key.Close()

	userPath, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return err
	}
	if pathListContains(userPath, dir) {
		return nil
	}
	var next string
	if strings.TrimSpace(userPath) == "" {
		next = dir
	} else {
		next = userPath + string(os.PathListSeparator) + dir
	}
	return key.SetStringValue("Path", next)
}
