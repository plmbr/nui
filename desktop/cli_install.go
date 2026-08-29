// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"nui/internal/update"
)

const (
	cliInstallMarker  = "desktop-cli.json"
	cliPathBlockBegin = "# >>> nui-desktop-cli >>>"
	cliPathBlockEnd   = "# <<< nui-desktop-cli <<<"
)

type desktopCLIState struct {
	Version     string `json:"version"`
	InstalledAt string `json:"installedAt"`
	Path        string `json:"path"`
}

// ensureCLIInstalled copies the bundled CGO-free nui CLI into the user install
// dir (same as install.sh / install.ps1) when missing or older than the
// bundled sidecar, and ensures that dir is on the user PATH. Never fatal —
// GUI launch continues on errors. Online upgrades of PATH CLI use GitHub
// Releases via the update API (pathCli target).
func ensureCLIInstalled() {
	if err := ensureCLIInstalledErr(); err != nil {
		fmt.Fprintf(os.Stderr, "nui desktop: CLI install: %v\n", err)
	}
}

func ensureCLIInstalledErr() error {
	bundled, err := bundledCLIPath()
	if err != nil {
		return err
	}
	if st, err := os.Stat(bundled); err != nil || st.IsDir() {
		// Dev launches (go run / wails dev) may not stage a sidecar.
		return nil
	}

	destDir, err := cliInstallDir()
	if err != nil {
		return err
	}
	dest := filepath.Join(destDir, cliBinaryName())

	if err := installBundledCLI(bundled, dest); err != nil {
		return err
	}

	if err := ensureCLIInstallDirOnPATH(destDir); err != nil {
		return fmt.Errorf("PATH setup: %w", err)
	}
	return nil
}

func installBundledCLI(bundled, dest string) error {
	bundledVer, err := readCLIVersion(bundled)
	if err != nil {
		return fmt.Errorf("read bundled version: %w", err)
	}

	if cliInstallDestExists(dest) {
		installedVer, err := readCLIVersion(dest)
		if err != nil {
			// Unreadable existing install — leave it alone.
			return nil
		}
		// Offline bump: replace PATH CLI only when the bundled sidecar is newer.
		if !isVersionNewer(bundledVer, installedVer) {
			return nil
		}
	}

	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	if err := copyFileAtomic(bundled, dest); err != nil {
		return fmt.Errorf("install %s: %w", dest, err)
	}
	_ = saveCLIState(desktopCLIState{
		Version:     bundledVer,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Path:        dest,
	})
	fmt.Fprintf(os.Stderr, "nui desktop: installed CLI %s -> %s\n", bundledVer, dest)
	return nil
}

func isVersionNewer(candidate, current string) bool {
	return update.IsNewer(candidate, current)
}

func cliInstallDestExists(dest string) bool {
	st, err := os.Stat(dest)
	return err == nil && !st.IsDir()
}

func cliBinaryName() string {
	if runtime.GOOS == "windows" {
		return "nui.exe"
	}
	return "nui"
}

func cliInstallDir() (string, error) {
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

func bundledCLIPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if runtime.GOOS == "darwin" {
		// nui.app/Contents/MacOS/nui-desktop → ../Resources/nui
		resources := filepath.Clean(filepath.Join(dir, "..", "Resources", "nui"))
		if st, err := os.Stat(resources); err == nil && !st.IsDir() {
			return resources, nil
		}
	}
	return filepath.Join(dir, cliBinaryName()), nil
}

func readCLIVersion(bin string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "version")
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func cliStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nui", cliInstallMarker), nil
}

func loadCLIState() (desktopCLIState, bool) {
	path, err := cliStatePath()
	if err != nil {
		return desktopCLIState{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return desktopCLIState{}, false
	}
	var st desktopCLIState
	if err := json.Unmarshal(data, &st); err != nil {
		return desktopCLIState{}, false
	}
	if strings.TrimSpace(st.Version) == "" {
		return desktopCLIState{}, false
	}
	return st, true
}

func saveCLIState(st desktopCLIState) error {
	path, err := cliStatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".nui-cli-*")
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
	if err := os.Rename(tmpName, dst); err != nil {
		// Windows may deny rename over existing file.
		_ = os.Remove(dst)
		if err2 := os.Rename(tmpName, dst); err2 != nil {
			return err
		}
	}
	return nil
}

func pathListContains(pathEnv, dir string) bool {
	dir = filepath.Clean(dir)
	for _, p := range filepath.SplitList(pathEnv) {
		if filepath.Clean(p) == dir {
			return true
		}
		if runtime.GOOS == "windows" && strings.EqualFold(filepath.Clean(p), dir) {
			return true
		}
	}
	return false
}
