// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeProcessEnvPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FROM_PROCESS", "process")
	t.Setenv("SHARED", "process-shared")
	_ = os.Unsetenv("FROM_SECRET")
	_ = os.Unsetenv("FROM_OVERRIDE")

	if err := SaveSecrets(Secrets{Env: map[string]string{
		"FROM_SECRET": "secret",
		"SHARED":      "secret-shared",
		"FROM_OVERRIDE": "secret-override",
	}}); err != nil {
		t.Fatal(err)
	}

	env := envMap(MergeProcessEnv(map[string]string{
		"FROM_OVERRIDE": "override",
		"NEW_KEY":       "new",
	}))
	if env["FROM_PROCESS"] != "process" {
		t.Fatalf("FROM_PROCESS = %q", env["FROM_PROCESS"])
	}
	if env["FROM_SECRET"] != "secret" {
		t.Fatalf("FROM_SECRET = %q", env["FROM_SECRET"])
	}
	if env["SHARED"] != "process-shared" {
		t.Fatalf("SHARED = %q (process should win over secret fill)", env["SHARED"])
	}
	if env["FROM_OVERRIDE"] != "override" {
		t.Fatalf("FROM_OVERRIDE = %q", env["FROM_OVERRIDE"])
	}
	if env["NEW_KEY"] != "new" {
		t.Fatalf("NEW_KEY = %q", env["NEW_KEY"])
	}
}

func TestExtensionProcessEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	_ = os.Unsetenv("GLOBAL_KEY")
	_ = os.Unsetenv("EXT_KEY")
	_ = os.Unsetenv("ADL_KEY")

	if err := SaveSecrets(Secrets{Env: map[string]string{"GLOBAL_KEY": "global"}}); err != nil {
		t.Fatal(err)
	}
	if err := SetExtensionEnv("demo", map[string]string{
		"EXT_KEY":    "ext",
		"GLOBAL_KEY": "ext-wins",
	}); err != nil {
		t.Fatal(err)
	}

	env := envMap(ExtensionProcessEnv("demo", map[string]string{"ADL_KEY": "adl", "EXT_KEY": "adl-wins"}))
	if env["GLOBAL_KEY"] != "ext-wins" {
		t.Fatalf("GLOBAL_KEY = %q", env["GLOBAL_KEY"])
	}
	if env["EXT_KEY"] != "adl-wins" {
		t.Fatalf("EXT_KEY = %q", env["EXT_KEY"])
	}
	if env["ADL_KEY"] != "adl" {
		t.Fatalf("ADL_KEY = %q", env["ADL_KEY"])
	}
}

func TestApplyGlobalEnvToProcess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".nui"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALREADY", "keep")
	_ = os.Unsetenv("FILL_ME")

	if err := SaveSecrets(Secrets{Env: map[string]string{
		"ALREADY": "ignore",
		"FILL_ME": "filled",
	}}); err != nil {
		t.Fatal(err)
	}
	ApplyGlobalEnvToProcess()
	if os.Getenv("ALREADY") != "keep" {
		t.Fatalf("ALREADY = %q", os.Getenv("ALREADY"))
	}
	if os.Getenv("FILL_ME") != "filled" {
		t.Fatalf("FILL_ME = %q", os.Getenv("FILL_ME"))
	}
}

func TestIsReservedEnvKey(t *testing.T) {
	if !IsReservedEnvKey("NUI_API_URL") {
		t.Fatal("expected NUI_API_URL reserved")
	}
	if !IsReservedEnvKey("NUI_EXTENSION_DIR") {
		t.Fatal("expected NUI_EXTENSION_DIR reserved")
	}
	if IsReservedEnvKey("MY_TOKEN") {
		t.Fatal("MY_TOKEN should not be reserved")
	}
	if IsReservedEnvKey("NUI_MY_EXTENSION_TOKEN") {
		t.Fatal("extension-owned NUI_ keys should be allowed")
	}
	if !IsReservedEnvKey("") {
		t.Fatal("empty key reserved")
	}
}

func envMap(environ []string) map[string]string {
	m := make(map[string]string, len(environ))
	for _, kv := range environ {
		k, v, ok := cutEnv(kv)
		if ok {
			m[k] = v
		}
	}
	return m
}

func cutEnv(kv string) (string, string, bool) {
	for i := 0; i < len(kv); i++ {
		if kv[i] == '=' {
			return kv[:i], kv[i+1:], true
		}
	}
	return "", "", false
}
