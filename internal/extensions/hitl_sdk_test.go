// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHitlSDKDirInstallsFiles(t *testing.T) {
	srcDir := filepath.Join("..", "..", "harness-sdk")
	if _, err := os.Stat(filepath.Join(srcDir, "loop_hitl.py")); err != nil {
		t.Skip("harness-sdk not found from test cwd")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LOOP_HITL_SDK_DIR", "")

	dir, err := installHitlSDK(srcDir)
	if err != nil {
		t.Fatalf("installHitlSDK: %v", err)
	}
	for _, name := range hitlSDKFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s in %s: %v", name, dir, err)
		}
	}
}
