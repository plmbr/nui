// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package harnesssdk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallDirWritesEmbeddedFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := InstallDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range FileNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s in %s: %v", name, dir, err)
		}
	}

	dir2, err := InstallDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir2 != dir {
		t.Fatalf("InstallDir() = %q, want %q on second call", dir2, dir)
	}
}

func TestReinstallDirOverwritesFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := InstallDir()
	if err != nil {
		t.Fatal(err)
	}
	stub := filepath.Join(dir, "nui_mention.py")
	if err := os.WriteFile(stub, []byte("# stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir2, err := ReinstallDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir2 != dir {
		t.Fatalf("ReinstallDir() = %q, want %q", dir2, dir)
	}
	data, err := os.ReadFile(stub)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "# stale\n" {
		t.Fatal("ReinstallDir did not overwrite nui_mention.py")
	}
}

func TestFilePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := FilePath("nui_mcp_tools.py")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
