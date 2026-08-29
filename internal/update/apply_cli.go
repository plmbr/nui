// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CLIInstallDir returns the default user install directory for the nui CLI
// (same as install.sh / desktop first-launch install).
func CLIInstallDir() (string, error) {
	if v := strings.TrimSpace(os.Getenv("NUI_INSTALL_DIR")); v != "" {
		return filepath.Clean(v), nil
	}
	if runtime.GOOS == "windows" {
		base := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "nui"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin"), nil
}

// CLIBinaryName is "nui" or "nui.exe".
func CLIBinaryName() string {
	if runtime.GOOS == "windows" {
		return "nui.exe"
	}
	return "nui"
}

// DefaultCLIPath is installDir/nui[.exe].
func DefaultCLIPath() (string, error) {
	dir, err := CLIInstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, CLIBinaryName()), nil
}

// CurrentExecutable resolves the path of the running binary.
func CurrentExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// ExtractCLIBinary unpacks a CLI release archive and returns the path to the
// extracted nui binary inside destDir.
func ExtractCLIBinary(archivePath, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	name := CLIBinaryName()
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		if err := extractTarGzFile(archivePath, destDir, name); err != nil {
			return "", err
		}
	case strings.HasSuffix(archivePath, ".zip"):
		if err := extractZipFile(archivePath, destDir, name); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported archive: %s", filepath.Base(archivePath))
	}
	out := filepath.Join(destDir, name)
	if st, err := os.Stat(out); err != nil || st.IsDir() {
		return "", fmt.Errorf("archive did not contain %s", name)
	}
	_ = os.Chmod(out, 0o755)
	return out, nil
}

// ApplyCLIBinary atomically replaces dest with the file at src.
func ApplyCLIBinary(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return replaceFile(src, dest)
}

// InstallCLIFromArchive extracts and installs the CLI to destPath.
func InstallCLIFromArchive(archivePath, destPath string) error {
	tmpDir, err := os.MkdirTemp("", "nui-cli-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	bin, err := ExtractCLIBinary(archivePath, tmpDir)
	if err != nil {
		return err
	}
	return ApplyCLIBinary(bin, destPath)
}

func replaceFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".nui-bin-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil && runtime.GOOS != "windows" {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		bak := dest + ".old"
		_ = os.Remove(bak)
		if _, err := os.Stat(dest); err == nil {
			if err := os.Rename(dest, bak); err != nil {
				return fmt.Errorf("rename running binary: %w", err)
			}
		}
		if err := os.Rename(tmpName, dest); err != nil {
			_ = os.Rename(bak, dest)
			return err
		}
		return nil
	}

	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(dest)
		if err2 := os.Rename(tmpName, dest); err2 != nil {
			return err
		}
	}
	return nil
}

func extractTarGzFile(archivePath, destDir, wantName string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s not found in archive", wantName)
		}
		if err != nil {
			return err
		}
		base := filepath.Base(hdr.Name)
		if base != wantName || hdr.Typeflag != tar.TypeReg {
			continue
		}
		outPath := filepath.Join(destDir, wantName)
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
}

func extractZipFile(archivePath, destDir, wantName string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if filepath.Base(f.Name) != wantName || f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		outPath := filepath.Join(destDir, wantName)
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	return fmt.Errorf("%s not found in archive", wantName)
}
