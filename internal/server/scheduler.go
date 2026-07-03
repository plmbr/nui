// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"loop/internal/model"
	"loop/internal/store"
)

var (
	schedulesMu sync.RWMutex
	schedules   []model.Schedule

	schedulerCancel context.CancelFunc
)

func loadSchedulesFromDisk() error {
	data, err := store.LoadSchedules()
	if err != nil {
		return err
	}
	schedulesMu.Lock()
	schedules = data.Schedules
	schedulesMu.Unlock()
	return nil
}

func persistSchedulesLocked() error {
	data := store.SchedulesData{Schedules: schedules}
	return store.SaveSchedules(data)
}

func listSchedulesCopy() []model.Schedule {
	schedulesMu.RLock()
	defer schedulesMu.RUnlock()
	out := make([]model.Schedule, len(schedules))
	copy(out, schedules)
	return out
}

func findSchedule(id string) (model.Schedule, bool) {
	schedulesMu.RLock()
	defer schedulesMu.RUnlock()
	for _, s := range schedules {
		if s.ID == id {
			return s, true
		}
	}
	return model.Schedule{}, false
}

func isAutoAgentType(agentType string) bool {
	def, ok := findADLDef(agentType)
	return ok && model.IsADLAutoPrompt(def)
}

func ensureScheduleNextRunAt(s *model.Schedule, from time.Time) error {
	if !s.Enabled {
		s.NextRunAt = ""
		return nil
	}
	if strings.TrimSpace(s.NextRunAt) != "" {
		if _, err := time.Parse(time.RFC3339, s.NextRunAt); err == nil {
			return nil
		}
	}
	next, err := s.ComputeNextRunAt(from)
	if err != nil {
		return err
	}
	s.NextRunAt = next.UTC().Format(time.RFC3339)
	return nil
}

func recoverScheduleNextRuns() {
	now := time.Now().UTC()
	schedulesMu.Lock()
	defer schedulesMu.Unlock()
	changed := false
	for i := range schedules {
		if !schedules[i].Enabled {
			continue
		}
		if err := ensureScheduleNextRunAt(&schedules[i], now); err != nil {
			fmt.Fprintf(os.Stderr, "warn: schedule %s next run: %v\n", schedules[i].ID, err)
			continue
		}
		changed = true
	}
	if changed {
		if err := persistSchedulesLocked(); err != nil {
			fmt.Fprintf(os.Stderr, "warn: save schedules on recovery: %v\n", err)
		}
	}
}

func startScheduler(ctx context.Context) {
	if err := loadSchedulesFromDisk(); err != nil {
		fmt.Fprintf(os.Stderr, "warn: load schedules: %v\n", err)
	}
	recoverScheduleNextRuns()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processDueSchedules()
		}
	}
}

func processDueSchedules() {
	now := time.Now().UTC()
	due := make([]model.Schedule, 0)
	schedulesMu.RLock()
	for _, s := range schedules {
		if !s.Enabled || strings.TrimSpace(s.NextRunAt) == "" {
			continue
		}
		next, err := time.Parse(time.RFC3339, s.NextRunAt)
		if err != nil {
			continue
		}
		if !now.Before(next) {
			due = append(due, s)
		}
	}
	schedulesMu.RUnlock()

	for _, s := range due {
		if err := fireSchedule(s.ID); err != nil {
			fmt.Fprintf(os.Stderr, "warn: schedule %s tick: %v\n", s.ID, err)
		}
	}
}

func fireSchedule(scheduleID string) error {
	schedulesMu.Lock()
	idx := -1
	for i, s := range schedules {
		if s.ID == scheduleID {
			idx = i
			break
		}
	}
	if idx < 0 {
		schedulesMu.Unlock()
		return fmt.Errorf("schedule not found")
	}
	s := schedules[idx]
	if !s.Enabled {
		schedulesMu.Unlock()
		return nil
	}

	if s.LastSessionID != "" && sessionHasRunningRun(s.LastSessionID) {
		next, err := s.ComputeNextRunAt(time.Now().UTC())
		if err != nil {
			schedulesMu.Unlock()
			return err
		}
		schedules[idx].NextRunAt = next.UTC().Format(time.RFC3339)
		if err := persistSchedulesLocked(); err != nil {
			schedulesMu.Unlock()
			return err
		}
		schedulesMu.Unlock()
		return nil
	}
	schedulesMu.Unlock()

	return triggerScheduleRun(scheduleID, false)
}

func triggerScheduleRun(scheduleID string, manual bool) error {
	schedulesMu.RLock()
	s, ok := findScheduleLocked(scheduleID)
	schedulesMu.RUnlock()
	if !ok {
		return fmt.Errorf("schedule not found")
	}

	def, ok := findADLDef(s.AgentType)
	if !ok || !model.IsADLAutoPrompt(def) {
		return fmt.Errorf("agent %q is not schedulable", s.AgentType)
	}

	now := time.Now().UTC()
	sessionName := strings.TrimSpace(s.Name)
	if sessionName == "" {
		sessionName = s.ID
	}
	session, err := createSessionEx(sessionCreateOpts{
		Name:         sessionName,
		WorkingDir:   s.WorkingDir,
		AgentType:    s.AgentType,
		ScheduleID:   s.ID,
		ScheduleName: s.Name,
	})
	if err != nil {
		return err
	}

	message := model.ResolveADLLaunchPrompt(def, s.Prompt)
	runID := uuid.NewString()
	createRunRecord(session.ID, runID, message)
	appendUserMessage(session.ID, message)

	mu.RLock()
	agentSessionID := agentSessions[session.ID]
	mu.RUnlock()

	go runInBackground(session.ID, session, agentSessionID, runID, message, false)

	schedulesMu.Lock()
	for i := range schedules {
		if schedules[i].ID != scheduleID {
			continue
		}
		schedules[i].LastRunAt = now.Format(time.RFC3339)
		schedules[i].LastSessionID = session.ID
		schedules[i].LastRunID = runID
		from := now
		if !manual {
			from = now
		}
		next, err := schedules[i].ComputeNextRunAt(from)
		if err != nil {
			schedulesMu.Unlock()
			return err
		}
		schedules[i].NextRunAt = next.UTC().Format(time.RFC3339)
		if err := persistSchedulesLocked(); err != nil {
			schedulesMu.Unlock()
			return err
		}
		break
	}
	schedulesMu.Unlock()
	return nil
}

func findScheduleLocked(id string) (model.Schedule, bool) {
	for _, s := range schedules {
		if s.ID == id {
			return s, true
		}
	}
	return model.Schedule{}, false
}

func runScheduler() {
	ctx, cancel := context.WithCancel(context.Background())
	schedulerCancel = cancel
	go startScheduler(ctx)
}

func stopScheduler() {
	if schedulerCancel != nil {
		schedulerCancel()
	}
}
