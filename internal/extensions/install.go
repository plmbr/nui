// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"archive/zip"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nui/internal/store"
)

// Install copies an extension from a git URL, local directory, or zip file into
// ~/.nui/extensions/<name>/.
func Install(source string) (string, error) {
	source = normalizeSource(source)
	if installType, _ := parsePackageSource(source); installType != "" {
		return installProgrammaticPackage(source)
	}
	if info, err := os.Stat(source); err == nil && info.IsDir() {
		if _, err := os.Stat(filepath.Join(source, manifestName)); os.IsNotExist(err) {
			if detectPackageType(source) != "" {
				return installProgrammaticFromDir(source)
			}
		}
	}
	root, cleanup, err := resolveInstallSource(source)
	if err != nil {
		return "", err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if _, err := os.Stat(filepath.Join(root, manifestName)); err != nil {
		if detectPackageType(root) != "" {
			return installProgrammaticFromDir(root)
		}
	}
	return installFromDir(root)
}

// Remove deletes an installed extension by id (manifest name).
func Remove(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("extension id is required")
	}
	if strings.Contains(name, string(os.PathSeparator)) || name == "." || name == ".." {
		return fmt.Errorf("invalid extension id %q", name)
	}
	extDir, err := store.ExtensionsDir()
	if err != nil {
		return err
	}
	target := filepath.Join(extDir, name)
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("extension %q is not installed", name)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("extension %q is not installed", name)
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	_ = store.RemoveExtensionEnv(name)
	return removeDisabledExtension(name)
}

func resolveInstallSource(source string) (root string, cleanup func(), err error) {
	source = normalizeSource(source)
	if source == "" {
		return "", nil, fmt.Errorf("source is required")
	}
	if isGitURL(source) {
		return cloneGitExtension(source)
	}
	info, err := os.Stat(source)
	if err != nil {
		return "", nil, fmt.Errorf("source %q: %w", source, err)
	}
	if info.IsDir() {
		root, err := findExtensionRoot(source)
		return root, nil, err
	}
	if strings.EqualFold(filepath.Ext(source), ".zip") {
		return extractZipExtension(source)
	}
	return "", nil, fmt.Errorf("source %q: expected a directory or .zip file", source)
}

func installFromDir(srcRoot string) (string, error) {
	manifest, err := loadManifestForInstall(srcRoot)
	if err != nil {
		return "", err
	}
	extDir, err := store.ExtensionsDir()
	if err != nil {
		return "", err
	}
	dst := filepath.Join(extDir, manifest.Name)
	if err := os.RemoveAll(dst); err != nil {
		return "", err
	}
	if err := copyDir(srcRoot, dst); err != nil {
		return "", err
	}
	if _, err := LoadManifest(dst); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	return manifest.Name, nil
}

func findExtensionRoot(dir string) (string, error) {
	dir = filepath.Clean(dir)
	if _, err := os.Stat(filepath.Join(dir, manifestName)); err == nil {
		return dir, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var matches []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(dir, e.Name())
		if _, err := os.Stat(filepath.Join(candidate, manifestName)); err == nil {
			matches = append(matches, candidate)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no %s found in %q", manifestName, dir)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple extensions found in %q; point at one extension directory", dir)
	}
}

func cloneGitExtension(gitURL string) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "nui-ext-clone-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	args := []string{"clone", "--depth", "1", gitURL, tmp}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("git clone: %w", err)
	}
	root, err := findExtensionRoot(tmp)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return root, cleanup, nil
}

func extractZipExtension(zipPath string) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "nui-ext-zip-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	tmp = filepath.Clean(tmp) + string(os.PathSeparator)
	for _, f := range r.File {
		target, err := safeZipTarget(tmp, f.Name)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				cleanup()
				return "", nil, err
			}
			continue
		}
		if err := extractZipFile(f, target); err != nil {
			cleanup()
			return "", nil, err
		}
	}
	root, err := findExtensionRoot(strings.TrimSuffix(tmp, string(os.PathSeparator)))
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return root, cleanup, nil
}

func safeZipTarget(destRoot, name string) (string, error) {
	target := filepath.Join(destRoot, name)
	target = filepath.Clean(target)
	if !strings.HasPrefix(target, destRoot) {
		return "", fmt.Errorf("zip entry %q escapes destination", name)
	}
	return target, nil
}

func extractZipFile(f *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return err
	}
	return out.Close()
}

func normalizeSource(source string) string {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "file://") {
		if u, err := url.Parse(source); err == nil && u.Path != "" {
			return u.Path
		}
		return strings.TrimPrefix(source, "file://")
	}
	if strings.HasPrefix(source, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, source[2:])
		}
	}
	return source
}

func isGitURL(source string) bool {
	switch {
	case strings.HasPrefix(source, "http://"), strings.HasPrefix(source, "https://"):
		return true
	case strings.HasPrefix(source, "git@"):
		return true
	case strings.HasPrefix(source, "git://"), strings.HasPrefix(source, "ssh://"):
		return true
	default:
		return false
	}
}

func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("copy dir: %q is not a directory", src)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func removeDisabledExtension(name string) error {
	settings, err := store.LoadSettings()
	if err != nil {
		return err
	}
	var next []string
	for _, ext := range settings.DisabledExtensions {
		if ext != name {
			next = append(next, ext)
		}
	}
	if len(next) == len(settings.DisabledExtensions) {
		return nil
	}
	settings.DisabledExtensions = next
	return store.SaveSettings(settings)
}
