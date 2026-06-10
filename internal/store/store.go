// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"loop/internal/model"
)

type Settings struct {
	Theme string `json:"theme"` // "light" | "dark"
}

type Data struct {
	Projects []model.Project   `json:"projects"`
	Sessions map[string]string `json:"sessions"`
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".loop")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func LoadSettings() (Settings, error) {
	dir, err := Dir()
	if err != nil {
		return Settings{Theme: "light"}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Settings{Theme: "light"}, nil
	}
	if err != nil {
		return Settings{Theme: "light"}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{Theme: "light"}, err
	}
	if s.Theme == "" {
		s.Theme = "light"
	}
	return s, nil
}

func SaveSettings(s Settings) error {
	return saveJSON("settings.json", s)
}

func LoadData() (Data, error) {
	empty := Data{Projects: []model.Project{}, Sessions: map[string]string{}}
	dir, err := Dir()
	if err != nil {
		return empty, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "data.json"))
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return empty, err
	}
	var d Data
	if err := json.Unmarshal(raw, &d); err != nil {
		return empty, err
	}
	if d.Projects == nil {
		d.Projects = []model.Project{}
	}
	if d.Sessions == nil {
		d.Sessions = map[string]string{}
	}
	return d, nil
}

func SaveData(d Data) error {
	return saveJSON("data.json", d)
}

func saveJSON(filename string, v any) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "*.tmp")
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
	return os.Rename(tmpPath, filepath.Join(dir, filename))
}
