// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"nui/internal/model"
)

func handleSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, listSchedulesCopy())
	case http.MethodPost:
		handleCreateSchedule(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSchedule(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/schedules/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(id, "/run-now") {
		id = strings.TrimSuffix(id, "/run-now")
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleRunScheduleNow(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s, ok := findSchedule(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, s)
	case http.MethodPatch:
		handlePatchSchedule(w, r, id)
	case http.MethodDelete:
		handleDeleteSchedule(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req model.Schedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.AgentType = strings.TrimSpace(req.AgentType)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.WorkingDir = strings.TrimSpace(req.WorkingDir)
	req.Interval = strings.TrimSpace(req.Interval)
	req.Cron = strings.TrimSpace(req.Cron)
	req.RunAt = strings.TrimSpace(req.RunAt)

	if err := model.ValidateScheduleInput(req, isAutoAgentType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	s := model.Schedule{
		ID:         uuid.NewString(),
		Name:       req.Name,
		AgentType:  req.AgentType,
		Prompt:     req.Prompt,
		WorkingDir: req.WorkingDir,
		Interval:   req.Interval,
		Cron:       req.Cron,
		RunAt:      req.RunAt,
		Enabled:    true,
		CreatedAt:  now.Format(time.RFC3339),
	}
	if err := ensureScheduleNextRunAt(&s, now); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	schedulesMu.Lock()
	schedules = append(schedules, s)
	if err := persistSchedulesLocked(); err != nil {
		schedules = schedules[:len(schedules)-1]
		schedulesMu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	schedulesMu.Unlock()

	writeJSON(w, http.StatusCreated, s)
}

func handlePatchSchedule(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Name       *string `json:"name"`
		AgentType  *string `json:"agentType"`
		Prompt     *string `json:"prompt"`
		WorkingDir *string `json:"workingDir"`
		Interval   *string `json:"interval"`
		Cron       *string `json:"cron"`
		RunAt      *string `json:"runAt"`
		Enabled    *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	schedulesMu.Lock()
	defer schedulesMu.Unlock()

	idx := -1
	for i, s := range schedules {
		if s.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		http.NotFound(w, r)
		return
	}

	s := schedules[idx]
	if req.Name != nil {
		s.Name = strings.TrimSpace(*req.Name)
	}
	if req.AgentType != nil {
		s.AgentType = strings.TrimSpace(*req.AgentType)
	}
	if req.Prompt != nil {
		s.Prompt = strings.TrimSpace(*req.Prompt)
	}
	if req.WorkingDir != nil {
		s.WorkingDir = strings.TrimSpace(*req.WorkingDir)
	}
	if req.Interval != nil {
		s.Interval = strings.TrimSpace(*req.Interval)
		s.Cron = ""
		s.RunAt = ""
	}
	if req.Cron != nil {
		s.Cron = strings.TrimSpace(*req.Cron)
		s.Interval = ""
		s.RunAt = ""
	}
	if req.RunAt != nil {
		s.RunAt = strings.TrimSpace(*req.RunAt)
		s.Interval = ""
		s.Cron = ""
	}
	if req.Enabled != nil {
		s.Enabled = *req.Enabled
	}

	if err := model.ValidateScheduleInput(s, isAutoAgentType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	if req.Interval != nil || req.Cron != nil || req.RunAt != nil || req.Enabled != nil {
		s.NextRunAt = ""
	}
	if err := ensureScheduleNextRunAt(&s, now); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	schedules[idx] = s
	if err := persistSchedulesLocked(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func handleDeleteSchedule(w http.ResponseWriter, r *http.Request, id string) {
	schedulesMu.Lock()
	defer schedulesMu.Unlock()

	idx := -1
	for i, s := range schedules {
		if s.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		http.NotFound(w, r)
		return
	}
	schedules = append(schedules[:idx], schedules[idx+1:]...)
	if err := persistSchedulesLocked(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleRunScheduleNow(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := findSchedule(id); !ok {
		http.NotFound(w, r)
		return
	}
	if err := triggerScheduleRun(id, true); err != nil {
		http.Error(w, fmt.Sprintf("run schedule: %v", err), http.StatusInternalServerError)
		return
	}
	s, _ := findSchedule(id)
	writeJSON(w, http.StatusAccepted, s)
}
