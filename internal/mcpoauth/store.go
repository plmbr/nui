// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpoauth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"loop/internal/store"
)

type storedCredential struct {
	Token      *oauth2.Token `json:"token"`
	ServerURL  string        `json:"serverUrl,omitempty"`
	ServerName string        `json:"serverName,omitempty"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

type tokenFile struct {
	Credentials map[string]storedCredential `json:"credentials"`
}

var storeMu sync.RWMutex
var tokensPathOverride string

// SetTokensPathForTest overrides the token store path (tests only).
func SetTokensPathForTest(path string) {
	storeMu.Lock()
	defer storeMu.Unlock()
	tokensPathOverride = path
}

func tokensPath() (string, error) {
	if tokensPathOverride != "" {
		return tokensPathOverride, nil
	}
	dir, err := store.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mcp-oauth", "tokens.json"), nil
}

func loadTokens() (tokenFile, error) {
	path, err := tokensPath()
	if err != nil {
		return tokenFile{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return tokenFile{Credentials: map[string]storedCredential{}}, nil
	}
	if err != nil {
		return tokenFile{}, err
	}
	var f tokenFile
	if err := json.Unmarshal(data, &f); err != nil {
		return tokenFile{}, err
	}
	if f.Credentials == nil {
		f.Credentials = map[string]storedCredential{}
	}
	return f, nil
}

func saveTokens(f tokenFile) error {
	path, err := tokensPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// SaveToken persists an OAuth token for a server key.
func SaveToken(key string, cred storedCredential) error {
	if key == "" {
		return errors.New("server key is required")
	}
	storeMu.Lock()
	defer storeMu.Unlock()
	f, err := loadTokens()
	if err != nil {
		return err
	}
	cred.UpdatedAt = time.Now().UTC()
	f.Credentials[key] = cred
	return saveTokens(f)
}

// LoadToken returns a stored credential for a server key.
func LoadToken(key string) (storedCredential, bool) {
	storeMu.RLock()
	defer storeMu.RUnlock()
	f, err := loadTokens()
	if err != nil {
		return storedCredential{}, false
	}
	cred, ok := f.Credentials[key]
	return cred, ok
}

// DeleteToken removes stored credentials for a server key.
func DeleteToken(key string) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	f, err := loadTokens()
	if err != nil {
		return err
	}
	delete(f.Credentials, key)
	return saveTokens(f)
}

// ListTokenKeys returns all server keys with stored credentials.
func ListTokenKeys() ([]string, error) {
	storeMu.RLock()
	defer storeMu.RUnlock()
	f, err := loadTokens()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(f.Credentials))
	for k := range f.Credentials {
		keys = append(keys, k)
	}
	return keys, nil
}

// HasValidToken reports whether a non-expired access token exists.
func HasValidToken(key string) bool {
	cred, ok := LoadToken(key)
	if !ok || cred.Token == nil || strings.TrimSpace(cred.Token.AccessToken) == "" {
		return false
	}
	if cred.Token.Expiry.IsZero() {
		return true
	}
	return cred.Token.Expiry.After(time.Now().Add(30 * time.Second))
}
