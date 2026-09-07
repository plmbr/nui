// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

//go:build darwin

package agent

import (
	"bytes"
	"os/exec"
)

const claudeKeychainService = "Claude Code-credentials"

func init() {
	readClaudeKeychainCredentials = func() ([]byte, error) {
		out, err := exec.Command("security", "find-generic-password",
			"-s", claudeKeychainService, "-w").Output()
		if err != nil {
			return nil, err
		}
		return bytes.TrimSpace(out), nil
	}
}
