// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestProgrammaticValidation(t *testing.T) {
	dir := t.TempDir()
	m := Manifest{
		Name: "prog-ext",
		Kind: "programmatic",
		Runtime: &RuntimeConfig{
			Transport: "stdio",
			Command:   []string{"python3", "${LOOP_EXTENSION_ENTRY}"},
		},
	}
	if err := validateManifest(dir, m, false); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestExpandRuntimeCommand(t *testing.T) {
	cmd := expandRuntimeCommand(
		[]string{"python3", "${LOOP_EXTENSION_ENTRY}"},
		"/ext",
		"/ext/pkg/host.py",
	)
	if cmd[0] != "python3" || cmd[1] != "/ext/pkg/host.py" {
		t.Fatalf("got %#v", cmd)
	}
}

func TestResolveInstallEntry(t *testing.T) {
	entry := resolveInstallEntry("/ext", &InstallConfig{
		Entry: "${LOOP_EXTENSION_DIR}/pkg/host.py",
	})
	want := filepath.Join("/ext", "pkg", "host.py")
	if entry != want {
		t.Fatalf("got %q want %q", entry, want)
	}
}

func TestParsePackageSource(t *testing.T) {
	typ, ref := parsePackageSource("npm:@corp/pkg@1.0.0")
	if typ != "npm" || ref != "@corp/pkg@1.0.0" {
		t.Fatalf("got %q %q", typ, ref)
	}
}

func TestInstallProgrammaticFromDir(t *testing.T) {
	src := filepath.Join("..", "..", "dev", "extension-examples", "programmatic-echo")
	if _, err := os.Stat(src); err != nil {
		t.Skip("example not present")
	}
	// Use temp extensions dir via store is hard; test metadata read only
	meta, err := readPackageMetadataFromDir(src)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	if meta.ID != "programmatic-echo" {
		t.Fatalf("id=%q", meta.ID)
	}
}
