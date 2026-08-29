// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const secretsFileName = "secrets.json"

// Secrets holds user-configured environment values: curated API provider
// credentials and free-form custom globals. Stored at ~/.nui/secrets.json
// with mode 0600 for desktop/app launches that do not inherit a shell
// environment. Applied to the nui process (fill blanks) and child processes.
type Secrets struct {
	Env map[string]string `json:"env"`
}

func emptySecrets() Secrets {
	return Secrets{Env: map[string]string{}}
}

func loadSecretsFile(dir string) (Secrets, error) {
	data, err := os.ReadFile(filepath.Join(dir, secretsFileName))
	if errors.Is(err, os.ErrNotExist) {
		return emptySecrets(), nil
	}
	if err != nil {
		return emptySecrets(), err
	}
	var s Secrets
	if err := json.Unmarshal(data, &s); err != nil {
		return emptySecrets(), err
	}
	if s.Env == nil {
		s.Env = map[string]string{}
	}
	return s, nil
}

// LoadUserSecrets reads secrets.json from the user data dir only.
func LoadUserSecrets() (Secrets, error) {
	dir, err := UserDir()
	if err != nil {
		return emptySecrets(), err
	}
	return loadSecretsFile(dir)
}

// LoadSystemSecrets reads secrets.json from the system config dir (if present).
func LoadSystemSecrets() (Secrets, error) {
	if !SystemDirExists() {
		return emptySecrets(), nil
	}
	return loadSecretsFile(SystemDir())
}

// LoadSecrets returns effective secrets: system base merged with user overrides.
func LoadSecrets() (Secrets, error) {
	sys, err := LoadSystemSecrets()
	if err != nil {
		sys = emptySecrets()
	}
	user, err := LoadUserSecrets()
	if err != nil {
		return Secrets{Env: mergeEnvMaps(sys.Env, nil)}, err
	}
	return Secrets{Env: mergeEnvMaps(sys.Env, user.Env)}, nil
}

// SaveSecrets writes secrets.json to the user data dir with mode 0600.
func SaveSecrets(s Secrets) error {
	if s.Env == nil {
		s.Env = map[string]string{}
	}
	cleaned := make(map[string]string, len(s.Env))
	for k, v := range s.Env {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		cleaned[k] = v
	}
	s.Env = cleaned

	dir, err := UserDir()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, secretsFileName)
	tmp, err := os.CreateTemp(dir, "secrets-*.tmp")
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

// SecretEnv returns the effective value for key (system + user), or empty if unset.
func SecretEnv(key string) string {
	s, err := LoadSecrets()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s.Env[key])
}
