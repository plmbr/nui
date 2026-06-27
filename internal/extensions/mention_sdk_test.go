// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"os"
	"path/filepath"
	"testing"

	"loop/internal/extensions"
)

func TestMentionSDKDirInstalls(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LOOP_MENTION_SDK_DIR", "")

	dir, err := extensions.MentionSDKDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "loop_mention.py")); err != nil {
		t.Fatalf("installed sdk: %v", err)
	}
}
