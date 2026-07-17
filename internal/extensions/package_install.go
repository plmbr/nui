// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nui/internal/store"

	"gopkg.in/yaml.v3"
)

const installLockName = "install.lock.json"

// InstallLock records how a programmatic extension was installed.
type InstallLock struct {
	Source      string `json:"source"`
	Type        string `json:"type"`
	Entry       string `json:"entry"`
	Root        string `json:"root,omitempty"`
	PackageName string `json:"packageName,omitempty"`
	NuiID       string `json:"nuiId,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Version     string `json:"version,omitempty"`
}

// PackageMetadata is read from npm/pip/go package manifests.
type PackageMetadata struct {
	ID          string
	DisplayName string
	Version     string
	Entry       string
	RuntimeCmd  []string
}

func installProgrammaticPackage(source string) (string, error) {
	installType, pkgRef := parsePackageSource(source)
	if installType == "" {
		return "", fmt.Errorf("unsupported package source %q (use npm:, pip:, or go:)", source)
	}
	meta, staging, cleanup, err := resolvePackageMetadata(installType, pkgRef, source)
	if err != nil {
		return "", err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if meta.ID == "" {
		return "", fmt.Errorf("package metadata: nui id is required")
	}
	extDir, err := store.ExtensionsDir()
	if err != nil {
		return "", err
	}
	dst := filepath.Join(extDir, meta.ID)
	if err := os.RemoveAll(dst); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return "", err
	}
	pkgRoot := filepath.Join(dst, "pkg")
	if err := installPackageArtifacts(installType, pkgRef, staging, pkgRoot); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	entry := meta.Entry
	if entry == "" {
		entry, err = detectInstalledEntry(installType, pkgRoot, meta)
		if err != nil {
			_ = os.RemoveAll(dst)
			return "", err
		}
	}
	entryRel := filepath.ToSlash(entry)
	if filepath.IsAbs(entry) {
		if rel, err := filepath.Rel(dst, entry); err == nil {
			entryRel = filepath.ToSlash(rel)
		}
	} else if !strings.HasPrefix(entryRel, "pkg/") {
		entryRel = filepath.ToSlash(filepath.Join("pkg", entryRel))
	}
	runtimeCmd := meta.RuntimeCmd
	if len(runtimeCmd) == 0 {
		runtimeCmd = defaultRuntimeCommand(installType)
	}
	manifest := Manifest{
		APIVersion:  "nui.plmbr.dev/extension/v1",
		Name:        meta.ID,
		Version:     meta.Version,
		DisplayName: meta.DisplayName,
		Kind:        "programmatic",
		Runtime: &RuntimeConfig{
			Transport: "stdio",
			Command:   runtimeCmd,
		},
		Install: &InstallConfig{
			Source: source,
			Type:   installType,
			Entry:  "${NUI_EXTENSION_DIR}/" + entryRel,
			Root:   "${NUI_EXTENSION_DIR}/pkg",
		},
	}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dst, manifestName), data, 0644); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	lock := InstallLock{
		Source:      source,
		Type:        installType,
		Entry:       manifest.Install.Entry,
		Root:        manifest.Install.Root,
		PackageName: pkgRef,
		NuiID:       meta.ID,
		DisplayName: meta.DisplayName,
		Version:     meta.Version,
	}
	lockData, _ := json.MarshalIndent(lock, "", "  ")
	if err := os.WriteFile(filepath.Join(dst, installLockName), lockData, 0644); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	if _, err := LoadManifest(dst); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	return meta.ID, nil
}

func parsePackageSource(source string) (installType, ref string) {
	source = strings.TrimSpace(source)
	switch {
	case strings.HasPrefix(source, "npm:"):
		return "npm", strings.TrimSpace(strings.TrimPrefix(source, "npm:"))
	case strings.HasPrefix(source, "pip:"):
		return "pip", strings.TrimSpace(strings.TrimPrefix(source, "pip:"))
	case strings.HasPrefix(source, "go:"):
		return "go", strings.TrimSpace(strings.TrimPrefix(source, "go:"))
	default:
		return "", source
	}
}

func defaultRuntimeCommand(installType string) []string {
	switch installType {
	case "npm":
		return []string{"node", "${NUI_EXTENSION_ENTRY}"}
	case "pip", "python":
		return []string{"python3", "${NUI_EXTENSION_ENTRY}"}
	default:
		return []string{"${NUI_EXTENSION_ENTRY}"}
	}
}

func detectPackageType(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		return "npm"
	}
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		return "pip"
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return "go"
	}
	return ""
}

func readPackageMetadataFromDir(dir string) (PackageMetadata, error) {
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		return readNPMPackageMetadata(dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		return readPyPackageMetadata(dir)
	}
	return PackageMetadata{}, fmt.Errorf("no supported package manifest in %q", dir)
}

func readNPMPackageMetadata(dir string) (PackageMetadata, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return PackageMetadata{}, err
	}
	var pkg struct {
		Name    string            `json:"name"`
		Version string            `json:"version"`
		Bin     map[string]string `json:"bin"`
		Nui struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			Entry       string `json:"entry"`
		} `json:"nui"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return PackageMetadata{}, err
	}
	meta := PackageMetadata{
		ID:          strings.TrimSpace(pkg.Nui.ID),
		DisplayName: strings.TrimSpace(pkg.Nui.DisplayName),
		Version:     strings.TrimSpace(pkg.Version),
	}
	if meta.DisplayName == "" {
		meta.DisplayName = pkg.Name
	}
	if entry := strings.TrimSpace(pkg.Nui.Entry); entry != "" {
		meta.Entry = filepath.Join(dir, entry)
	} else if len(pkg.Bin) > 0 {
		for _, path := range pkg.Bin {
			meta.Entry = filepath.Join(dir, path)
			break
		}
	}
	return meta, nil
}

func readPyPackageMetadata(dir string) (PackageMetadata, error) {
	data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		return PackageMetadata{}, err
	}
	meta := PackageMetadata{}
	lines := strings.Split(string(data), "\n")
	inToolnui := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "[tool.nui]" {
			inToolnui = true
			continue
		}
		if strings.HasPrefix(trim, "[") && trim != "[tool.nui]" {
			inToolnui = false
		}
		if !inToolnui {
			continue
		}
		if strings.HasPrefix(trim, "id = ") {
			meta.ID = strings.Trim(strings.TrimPrefix(trim, "id = "), `"`)
		}
		if strings.HasPrefix(trim, "displayName = ") {
			meta.DisplayName = strings.Trim(strings.TrimPrefix(trim, "displayName = "), `"`)
		}
	}
	return meta, nil
}

func resolvePackageMetadata(installType, pkgRef, source string) (PackageMetadata, string, func(), error) {
	switch installType {
	case "npm", "pip":
		staging, cleanup, err := stagePackageSource(installType, pkgRef)
		if err != nil {
			return PackageMetadata{}, "", nil, err
		}
		meta, err := readPackageMetadataFromDir(staging)
		return meta, staging, cleanup, err
	case "go":
		return PackageMetadata{ID: goModuleID(pkgRef), DisplayName: goModuleID(pkgRef)}, "", nil, fmt.Errorf("go package install not yet implemented for %q", pkgRef)
	default:
		return PackageMetadata{}, "", nil, fmt.Errorf("unsupported install type %q", installType)
	}
}

func goModuleID(ref string) string {
	ref = strings.TrimSpace(ref)
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		ref = ref[i+1:]
	}
	if i := strings.Index(ref, "@"); i >= 0 {
		ref = ref[:i]
	}
	return ref
}

func stagePackageSource(installType, pkgRef string) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "nui-ext-stage-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	switch installType {
	case "npm":
		cmd := exec.Command("npm", "pack", pkgRef, "--pack-destination", tmp)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("npm pack: %w", err)
		}
		entries, _ := os.ReadDir(tmp)
		var tgz string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".tgz") {
				tgz = filepath.Join(tmp, e.Name())
				break
			}
		}
		if tgz == "" {
			cleanup()
			return "", nil, fmt.Errorf("npm pack produced no .tgz")
		}
		extractDir := filepath.Join(tmp, "extract")
		if err := os.MkdirAll(extractDir, 0755); err != nil {
			cleanup()
			return "", nil, err
		}
		cmd = exec.Command("tar", "-xzf", tgz, "-C", extractDir)
		if err := cmd.Run(); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("extract npm pack: %w", err)
		}
		pkgDir := filepath.Join(extractDir, "package")
		if _, err := os.Stat(pkgDir); err != nil {
			cleanup()
			return "", nil, err
		}
		return pkgDir, cleanup, nil
	case "pip":
		return tmp, cleanup, nil
	default:
		cleanup()
		return "", nil, fmt.Errorf("unsupported stage type %q", installType)
	}
}

func installPackageArtifacts(installType, pkgRef, staging, pkgRoot string) error {
	switch installType {
	case "npm":
		cmd := exec.Command("npm", "install", "--omit=dev", "--prefix", pkgRoot, staging)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "pip":
		cmd := exec.Command("pip", "install", pkgRef, "--target", pkgRoot)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported install type %q", installType)
	}
}

func detectInstalledEntry(installType, pkgRoot string, meta PackageMetadata) (string, error) {
	if meta.Entry != "" {
		if filepath.IsAbs(meta.Entry) {
			return meta.Entry, nil
		}
		return filepath.Join(pkgRoot, meta.Entry), nil
	}
	if installType == "npm" {
		nodeModules := filepath.Join(pkgRoot, "node_modules")
		entries, err := os.ReadDir(nodeModules)
		if err != nil {
			return "", err
		}
		for _, e := range entries {
			if e.Name() == ".bin" {
				continue
			}
			pkgDir := filepath.Join(nodeModules, e.Name())
			if _, err := os.Stat(filepath.Join(pkgDir, "package.json")); err == nil {
				m, err := readNPMPackageMetadata(pkgDir)
				if err == nil && m.Entry != "" {
					return m.Entry, nil
				}
			}
		}
	}
	return "", fmt.Errorf("could not detect package entry in %q", pkgRoot)
}

func installProgrammaticFromDir(srcRoot string) (string, error) {
	meta, err := readPackageMetadataFromDir(srcRoot)
	if err != nil {
		return "", err
	}
	if meta.ID == "" {
		return "", fmt.Errorf("package metadata: nui id is required in %q", srcRoot)
	}
	extDir, err := store.ExtensionsDir()
	if err != nil {
		return "", err
	}
	dst := filepath.Join(extDir, meta.ID)
	if err := os.RemoveAll(dst); err != nil {
		return "", err
	}
	if err := copyDir(srcRoot, filepath.Join(dst, "pkg")); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	pkgType := detectPackageType(srcRoot)
	if pkgType == "pip" {
		pkgType = "python"
	}
	entry := meta.Entry
	if entry == "" {
		entry = filepath.Join(dst, "pkg", "host.py")
		if _, err := os.Stat(entry); err != nil {
			entry, err = detectInstalledEntry(pkgType, filepath.Join(dst, "pkg"), meta)
			if err != nil {
				_ = os.RemoveAll(dst)
				return "", err
			}
		}
	}
	entryRel := filepath.ToSlash(entry)
	if filepath.IsAbs(entry) {
		if rel, err := filepath.Rel(dst, entry); err == nil {
			entryRel = filepath.ToSlash(rel)
		}
	}
	manifest := Manifest{
		APIVersion:  "nui.plmbr.dev/extension/v1",
		Name:        meta.ID,
		Version:     meta.Version,
		DisplayName: meta.DisplayName,
		Kind:        "programmatic",
		Runtime: &RuntimeConfig{
			Transport: "stdio",
			Command:   defaultRuntimeCommand(pkgType),
		},
		Install: &InstallConfig{
			Source: srcRoot,
			Type:   "dir",
			Entry:  "${NUI_EXTENSION_DIR}/" + entryRel,
			Root:   "${NUI_EXTENSION_DIR}/pkg",
		},
	}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dst, manifestName), data, 0644); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	if _, err := LoadManifest(dst); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	return meta.ID, nil
}

// DeliverExtensionHITL routes a HITL request to extension channels via programmatic hosts.
func (r *Registry) DeliverExtensionHITL(channelRef string, request map[string]any, workingDir, sessionID string) error {
	extName, channelID, ok := ParseExtRef(channelRef)
	if !ok {
		return fmt.Errorf("invalid channel ref %q", channelRef)
	}
	r.mu.RLock()
	ext, ok := r.extensions[extName]
	r.mu.RUnlock()
	if !ok || r.isDisabled(extName) {
		return fmt.Errorf("extension %q not found", extName)
	}
	if ext.programmaticHost == nil {
		return fmt.Errorf("extension %q has no programmatic host for HITL delivery", extName)
	}
	return ext.programmaticHost.DeliverHITL(channelID, request, workingDir, sessionID)
}
