// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"loop/internal/model"
)

type SchedulesData struct {
	Schedules []model.Schedule `json:"schedules"`
}

func LoadSchedules() (SchedulesData, error) {
	empty := SchedulesData{Schedules: []model.Schedule{}}
	dir, err := Dir()
	if err != nil {
		return empty, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "schedules.json"))
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return empty, err
	}
	var d SchedulesData
	if err := json.Unmarshal(raw, &d); err != nil {
		return empty, err
	}
	if d.Schedules == nil {
		d.Schedules = []model.Schedule{}
	}
	return d, nil
}

func SaveSchedules(d SchedulesData) error {
	if d.Schedules == nil {
		d.Schedules = []model.Schedule{}
	}
	return saveJSON("schedules.json", d)
}
