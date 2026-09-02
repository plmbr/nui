// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"nui/internal/extensions"
)

func TestInstallFromDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	src := filepath.Join("..", "..", "dev", "extension-examples", "corp-pack")
	name, err := extensions.Install(src, true)
	if err != nil {
		t.Fatal(err)
	}
	if name != "corp-pack" {
		t.Fatalf("name: got %q want corp-pack", name)
	}

	dst := filepath.Join(home, ".nui", "extensions", "corp-pack", "extension.yaml")
	if _, err := os.Stat(dst); err != nil {
		t.Fatal(err)
	}

	reg, err := extensions.LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Get("corp-pack"); !ok {
		t.Fatal("corp-pack not loaded")
	}
}

func TestInstallFromZip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	srcDir := filepath.Join("..", "..", "dev", "extension-examples", "corp-pack")
	zipPath := filepath.Join(t.TempDir(), "corp-pack.zip")
	if err := zipDir(srcDir, zipPath); err != nil {
		t.Fatal(err)
	}

	name, err := extensions.Install(zipPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if name != "corp-pack" {
		t.Fatalf("name: got %q want corp-pack", name)
	}
}

func TestInstallAndRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	src := filepath.Join("..", "..", "dev", "extension-examples", "corp-pack")
	if _, err := extensions.Install(src, true); err != nil {
		t.Fatal(err)
	}
	if err := extensions.Remove("corp-pack"); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(home, ".nui", "extensions", "corp-pack")
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("expected corp-pack removed, stat err=%v", err)
	}
}

func TestRemoveMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := extensions.Remove("missing"); err == nil {
		t.Fatal("expected error removing missing extension")
	}
}

func zipDir(srcDir, zipPath string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		f, err := w.Create(rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	})
}
