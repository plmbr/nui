// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"os"
	"path/filepath"
	"testing"

	"nui/internal/extensions"
)

func TestMentionSDKDirInstalls(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NUI_MENTION_SDK_DIR", "")

	dir, err := extensions.MentionSDKDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "nui_mention.py")); err != nil {
		t.Fatalf("installed sdk: %v", err)
	}
}
