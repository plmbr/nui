// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package devcontainer

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nui/internal/devcontainer/dockercontext"
)

// dockerContextDirs maps inner harness types to docker/ build context directory names.
var dockerContextDirs = map[string]string{
	"claude-code": "devcontainer-claude-code",
	"pi":          "devcontainer-pi",
	"codex":       "devcontainer-codex",
	"opencode":    "devcontainer-opencode",
}

// ResolveImage returns the image tag for a devcontainer harness.
func ResolveImage(innerHarness, imageOverride string) (string, error) {
	inner := strings.TrimSpace(innerHarness)
	if inner == "" {
		return "", fmt.Errorf("inner harness is required")
	}
	image := strings.TrimSpace(imageOverride)
	if image == "" {
		image = DefaultImages[inner]
	}
	if image == "" {
		return "", fmt.Errorf("no default devcontainer image for inner harness %q", inner)
	}
	return image, nil
}

// IsNuiManagedImage reports whether nui can auto-build the requested image for innerHarness.
func IsNuiManagedImage(innerHarness, image string) bool {
	defaultImage, err := ResolveImage(innerHarness, "")
	if err != nil {
		return false
	}
	return strings.TrimSpace(image) == defaultImage
}

// ImageExists reports whether a Docker image tag exists locally.
func ImageExists(image string) bool {
	image = strings.TrimSpace(image)
	if image == "" {
		return false
	}
	cmd := exec.Command("docker", "image", "inspect", image)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

// EnsureImage builds the nui-managed devcontainer image when it is missing locally.
// Custom image overrides are not auto-built.
func EnsureImage(ctx context.Context, innerHarness, imageOverride string) error {
	if !DockerAvailable() {
		return fmt.Errorf("Docker is not running or not reachable. Start Docker Desktop and retry")
	}

	image, err := ResolveImage(innerHarness, imageOverride)
	if err != nil {
		return err
	}
	if ImageExists(image) {
		return nil
	}
	if !IsNuiManagedImage(innerHarness, image) {
		return fmt.Errorf("devcontainer image %q not found locally (custom image override; build or pull it manually)", image)
	}

	contextDir, err := resolveBuildContext(innerHarness)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "[nui] building devcontainer image %s (first use; this may take a few minutes)\n", image)
	if err := buildImage(ctx, image, contextDir); err != nil {
		return err
	}
	if !ImageExists(image) {
		return fmt.Errorf("devcontainer image %q still missing after build", image)
	}
	fmt.Fprintf(os.Stderr, "[nui] devcontainer image %s ready\n", image)
	return nil
}

func buildImage(ctx context.Context, image, contextDir string) error {
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", image, contextDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build -t %s %s: %w", image, contextDir, err)
	}
	return nil
}

func resolveBuildContext(innerHarness string) (string, error) {
	if dir, err := findRepoBuildContext(innerHarness); err == nil {
		return dir, nil
	}
	return materializeEmbeddedBuildContext(innerHarness)
}

func findRepoBuildContext(innerHarness string) (string, error) {
	contextName, ok := dockerContextDirs[strings.TrimSpace(innerHarness)]
	if !ok {
		return "", fmt.Errorf("unknown inner harness %q", innerHarness)
	}
	rel := filepath.Join("docker", contextName)
	for _, start := range searchRoots() {
		dir := start
		for i := 0; i < 8; i++ {
			candidate := filepath.Join(dir, rel)
			if st, err := os.Stat(filepath.Join(candidate, "Dockerfile")); err == nil && !st.IsDir() {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "", fmt.Errorf("docker build context not found in repo")
}

func materializeEmbeddedBuildContext(innerHarness string) (string, error) {
	contextName, ok := dockerContextDirs[strings.TrimSpace(innerHarness)]
	if !ok {
		return "", fmt.Errorf("unknown inner harness %q", innerHarness)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".nui", "cache", "devcontainer-build", contextName)
	dockerfile := filepath.Join(dir, "Dockerfile")
	if st, err := os.Stat(dockerfile); err == nil && !st.IsDir() {
		return dir, nil
	}

	data, err := fs.ReadFile(dockercontext.FS, filepath.Join(contextName, "Dockerfile"))
	if err != nil {
		return "", fmt.Errorf("read embedded Dockerfile for %q: %w", contextName, err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dockerfile, data, 0644); err != nil {
		return "", err
	}
	return dir, nil
}

func searchRoots() []string {
	var roots []string
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	return roots
}
