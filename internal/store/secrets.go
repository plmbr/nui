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

// Secrets holds user-configured environment values for API providers.
// Stored at ~/.nui/secrets.json with mode 0600 for desktop/app launches
// that do not inherit a shell environment.
type Secrets struct {
	Env map[string]string `json:"env"`
}

// LoadSecrets reads ~/.nui/secrets.json. Missing file yields empty Env.
func LoadSecrets() (Secrets, error) {
	dir, err := Dir()
	if err != nil {
		return Secrets{Env: map[string]string{}}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, secretsFileName))
	if errors.Is(err, os.ErrNotExist) {
		return Secrets{Env: map[string]string{}}, nil
	}
	if err != nil {
		return Secrets{Env: map[string]string{}}, err
	}
	var s Secrets
	if err := json.Unmarshal(data, &s); err != nil {
		return Secrets{Env: map[string]string{}}, err
	}
	if s.Env == nil {
		s.Env = map[string]string{}
	}
	return s, nil
}

// SaveSecrets writes ~/.nui/secrets.json with mode 0600.
func SaveSecrets(s Secrets) error {
	if s.Env == nil {
		s.Env = map[string]string{}
	}
	// Drop empty values so the file stays tidy.
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

	dir, err := Dir()
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

// SecretEnv returns the value stored for key, or empty if unset.
func SecretEnv(key string) string {
	s, err := LoadSecrets()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s.Env[key])
}
