// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHarnessSDKReinstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"harness-sdk", "reinstall"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(home, ".nui", "harness-sdk")
	if _, err := os.Stat(filepath.Join(dir, "nui_mcp_tools.py")); err != nil {
		t.Fatalf("reinstall did not write sdk files: %v", err)
	}
}
