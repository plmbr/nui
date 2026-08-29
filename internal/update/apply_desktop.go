// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DesktopInstallRoot returns the directory that contains the running desktop
// app (the .app bundle on macOS, or the folder with nui-desktop on Win/Linux).
func DesktopInstallRoot() (string, error) {
	exe, err := CurrentExecutable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if runtime.GOOS == "darwin" {
		// .../nui.app/Contents/MacOS → .../nui.app
		if strings.HasSuffix(dir, filepath.Join("Contents", "MacOS")) {
			return filepath.Clean(filepath.Join(dir, "..", "..")), nil
		}
	}
	return dir, nil
}

// ApplyDesktopFromArchive extracts a desktop release archive and replaces the
// current installation at installRoot. installRoot is the .app path on darwin
// or the directory containing nui-desktop on windows/linux.
func ApplyDesktopFromArchive(archivePath, installRoot string) error {
	tmpDir, err := os.MkdirTemp("", "nui-desktop-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		if err := extractAllTarGz(archivePath, tmpDir); err != nil {
			return err
		}
	case strings.HasSuffix(archivePath, ".zip"):
		if err := extractAllZip(archivePath, tmpDir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported desktop archive: %s", filepath.Base(archivePath))
	}

	switch runtime.GOOS {
	case "darwin":
		return applyDesktopDarwin(tmpDir, installRoot)
	case "windows":
		return applyDesktopWindows(tmpDir, installRoot)
	default:
		return applyDesktopUnix(tmpDir, installRoot)
	}
}

// RelaunchDesktop starts a new desktop process and returns so the caller can exit.
func RelaunchDesktop(exePath string) error {
	cmd := exec.Command(exePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Start()
}

// DesktopExecutablePath returns the path to launch after an update.
func DesktopExecutablePath(installRoot string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(installRoot, "Contents", "MacOS", "nui-desktop")
	case "windows":
		p := filepath.Join(installRoot, "nui-desktop.exe")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
		return filepath.Join(installRoot, "nui.exe")
	default:
		p := filepath.Join(installRoot, "nui-desktop")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
		return filepath.Join(installRoot, "nui")
	}
}

func applyDesktopDarwin(extractDir, appPath string) error {
	srcApp := findAppBundle(extractDir)
	if srcApp == "" {
		return fmt.Errorf("archive did not contain nui.app")
	}
	parent := filepath.Dir(appPath)
	bak := appPath + ".bak"
	_ = os.RemoveAll(bak)
	if _, err := os.Stat(appPath); err == nil {
		if err := os.Rename(appPath, bak); err != nil {
			return fmt.Errorf("backup current app: %w", err)
		}
	}
	dest := filepath.Join(parent, filepath.Base(srcApp))
	if err := copyDir(srcApp, dest); err != nil {
		_ = os.Rename(bak, appPath)
		return err
	}
	_ = os.RemoveAll(bak)
	// Clear quarantine on the replaced app when possible.
	if _, err := exec.LookPath("xattr"); err == nil {
		_ = exec.Command("xattr", "-cr", dest).Run()
	}
	return nil
}

func applyDesktopWindows(extractDir, installDir string) error {
	return replaceDesktopBins(extractDir, installDir, []string{"nui-desktop.exe", "nui.exe"})
}

func applyDesktopUnix(extractDir, installDir string) error {
	return replaceDesktopBins(extractDir, installDir, []string{"nui-desktop", "nui"})
}

func replaceDesktopBins(extractDir, installDir string, names []string) error {
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}
	found := false
	for _, name := range names {
		src := findFileNamed(extractDir, name)
		if src == "" {
			continue
		}
		found = true
		dest := filepath.Join(installDir, name)
		if err := replaceFile(src, dest); err != nil {
			return fmt.Errorf("replace %s: %w", name, err)
		}
	}
	if !found {
		return fmt.Errorf("archive did not contain desktop binaries")
	}
	return nil
}

func findAppBundle(root string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if d.Name() == "nui.app" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func findFileNamed(root, name string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func copyDir(src, dest string) error {
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("ditto"); err == nil {
			cmd := exec.Command("ditto", src, dest)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("ditto: %w: %s", err, strings.TrimSpace(string(out)))
			}
			return nil
		}
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFileMode(path, target)
	})
}

func copyFileMode(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, st.Mode())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func extractAllTarGz(archivePath, destDir string) error {
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
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, filepath.Clean(hdr.Name))
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) && target != filepath.Clean(destDir) {
			return fmt.Errorf("invalid tar path: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)|0o755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

func extractAllZip(archivePath, destDir string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		target := filepath.Join(destDir, filepath.Clean(f.Name))
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) && target != filepath.Clean(destDir) {
			return fmt.Errorf("invalid zip path: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		mode := f.Mode()
		if mode&0o111 != 0 || runtime.GOOS != "windows" {
			mode |= 0o755
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
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
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
