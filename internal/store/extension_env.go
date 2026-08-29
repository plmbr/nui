// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const extensionEnvFileName = "extension-env.json"

// ExtensionEnvFile holds per-extension environment maps.
// Stored at ~/.nui/extension-env.json with mode 0600.
type ExtensionEnvFile struct {
	Env map[string]map[string]string `json:"env"`
}

func emptyExtensionEnv() ExtensionEnvFile {
	return ExtensionEnvFile{Env: map[string]map[string]string{}}
}

func loadExtensionEnvFile(dir string) (ExtensionEnvFile, error) {
	data, err := os.ReadFile(filepath.Join(dir, extensionEnvFileName))
	if errors.Is(err, os.ErrNotExist) {
		return emptyExtensionEnv(), nil
	}
	if err != nil {
		return emptyExtensionEnv(), err
	}
	var f ExtensionEnvFile
	if err := json.Unmarshal(data, &f); err != nil {
		return emptyExtensionEnv(), err
	}
	if f.Env == nil {
		f.Env = map[string]map[string]string{}
	}
	return f, nil
}

// LoadUserExtensionEnv reads extension-env.json from the user data dir only.
func LoadUserExtensionEnv() (ExtensionEnvFile, error) {
	dir, err := UserDir()
	if err != nil {
		return emptyExtensionEnv(), err
	}
	return loadExtensionEnvFile(dir)
}

// LoadSystemExtensionEnv reads extension-env.json from the system config dir.
func LoadSystemExtensionEnv() (ExtensionEnvFile, error) {
	if !SystemDirExists() {
		return emptyExtensionEnv(), nil
	}
	return loadExtensionEnvFile(SystemDir())
}

// LoadExtensionEnv returns effective extension env (system base + user overrides).
func LoadExtensionEnv() (ExtensionEnvFile, error) {
	sys, err := LoadSystemExtensionEnv()
	if err != nil {
		sys = emptyExtensionEnv()
	}
	user, err := LoadUserExtensionEnv()
	if err != nil {
		return ExtensionEnvFile{Env: mergeExtensionEnvMaps(sys.Env, nil)}, err
	}
	return ExtensionEnvFile{Env: mergeExtensionEnvMaps(sys.Env, user.Env)}, nil
}

func mergeExtensionEnvMaps(sys, user map[string]map[string]string) map[string]map[string]string {
	out := map[string]map[string]string{}
	for name, env := range sys {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[name] = cleanEnvMap(env)
	}
	for name, env := range user {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[name] = mergeEnvMaps(out[name], cleanEnvMap(env))
	}
	return out
}

// SaveExtensionEnv writes extension-env.json to the user data dir with mode 0600.
func SaveExtensionEnv(f ExtensionEnvFile) error {
	if f.Env == nil {
		f.Env = map[string]map[string]string{}
	}
	cleaned := make(map[string]map[string]string, len(f.Env))
	for name, env := range f.Env {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		inner := cleanEnvMap(env)
		if len(inner) == 0 {
			continue
		}
		cleaned[name] = inner
	}
	f.Env = cleaned

	dir, err := UserDir()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, extensionEnvFileName)
	tmp, err := os.CreateTemp(dir, "extension-env-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_, werr := tmp.Write(b)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	if werr != nil {
		os.Remove(tmpPath)
		return werr
	}
	if syncErr != nil {
		os.Remove(tmpPath)
		return syncErr
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return closeErr
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	_ = os.Chmod(path, 0o600)
	return nil
}

// ExtensionEnv returns the effective env map for one extension (may be empty).
func ExtensionEnv(name string) map[string]string {
	name = strings.TrimSpace(name)
	if name == "" {
		return map[string]string{}
	}
	f, err := LoadExtensionEnv()
	if err != nil {
		return map[string]string{}
	}
	return cleanEnvMap(f.Env[name])
}

// ExtensionEnvKeys returns sorted key names for an extension (values omitted).
func ExtensionEnvKeys(name string) []string {
	env := ExtensionEnv(name)
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SetExtensionEnv replaces the user-layer env map for one extension.
// Empty map deletes the user entry (system defaults remain effective).
func SetExtensionEnv(name string, env map[string]string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("extension name is required")
	}
	f, err := LoadUserExtensionEnv()
	if err != nil {
		return err
	}
	if f.Env == nil {
		f.Env = map[string]map[string]string{}
	}
	cleaned := cleanEnvMap(env)
	if len(cleaned) == 0 {
		delete(f.Env, name)
	} else {
		f.Env[name] = cleaned
	}
	return SaveExtensionEnv(f)
}

// RemoveExtensionEnv deletes the user-layer env entry for an extension.
func RemoveExtensionEnv(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	f, err := LoadUserExtensionEnv()
	if err != nil {
		return err
	}
	if f.Env == nil {
		return nil
	}
	if _, ok := f.Env[name]; !ok {
		return nil
	}
	delete(f.Env, name)
	return SaveExtensionEnv(f)
}

func cleanEnvMap(env map[string]string) map[string]string {
	if len(env) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" || IsReservedEnvKey(k) {
			continue
		}
		out[k] = v
	}
	return out
}
