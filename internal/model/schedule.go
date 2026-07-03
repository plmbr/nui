// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var scheduleDaySuffix = regexp.MustCompile(`^(\d+(?:\.\d+)?)d$`)
var scheduleWeekSuffix = regexp.MustCompile(`^(\d+(?:\.\d+)?)w$`)

// ParseScheduleInterval parses Go duration strings plus day (d) and week (w) suffixes.
func ParseScheduleInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty interval")
	}
	if m := scheduleDaySuffix.FindStringSubmatch(s); m != nil {
		n, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid interval: %w", err)
		}
		return time.Duration(n * float64(24*time.Hour)), nil
	}
	if m := scheduleWeekSuffix.FindStringSubmatch(s); m != nil {
		n, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid interval: %w", err)
		}
		return time.Duration(n * float64(7*24*time.Hour)), nil
	}
	return time.ParseDuration(s)
}

const MinScheduleInterval = 30 * time.Second

// Schedule describes a recurring autonomous run for a promptMode:auto agent.
type Schedule struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AgentType     string `json:"agentType"`
	Prompt        string `json:"prompt,omitempty"`
	WorkingDir    string `json:"workingDir,omitempty"`
	Interval      string `json:"interval,omitempty"`
	Cron          string `json:"cron,omitempty"`
	Enabled       bool   `json:"enabled"`
	LastRunAt     string `json:"lastRunAt,omitempty"`
	NextRunAt     string `json:"nextRunAt,omitempty"`
	LastSessionID string `json:"lastSessionId,omitempty"`
	LastRunID     string `json:"lastRunId,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

// ValidateScheduleInput checks schedule fields before create/update.
func ValidateScheduleInput(s Schedule, isAutoAgent func(agentType string) bool) error {
	name := strings.TrimSpace(s.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	agentType := strings.TrimSpace(s.AgentType)
	if agentType == "" {
		return fmt.Errorf("agentType is required")
	}
	if isAutoAgent != nil && !isAutoAgent(agentType) {
		return fmt.Errorf("agent %q must use promptMode auto to be scheduled", agentType)
	}

	hasInterval := strings.TrimSpace(s.Interval) != ""
	hasCron := strings.TrimSpace(s.Cron) != ""
	if hasInterval == hasCron {
		return fmt.Errorf("exactly one of interval or cron is required")
	}
	if hasInterval {
		d, err := ParseScheduleInterval(s.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		if d < MinScheduleInterval {
			return fmt.Errorf("interval must be at least %s", MinScheduleInterval)
		}
	}
	if hasCron {
		if _, err := cron.ParseStandard(strings.TrimSpace(s.Cron)); err != nil {
			return fmt.Errorf("invalid cron: %w", err)
		}
	}
	return nil
}

// nextIntervalBoundary returns the next UTC instant strictly after from on the
// interval grid anchored at the Unix epoch (e.g. every 5m at :00, :05, :10…).
func nextIntervalBoundary(from time.Time, interval time.Duration) time.Time {
	from = from.UTC()
	step := interval.Nanoseconds()
	if step <= 0 {
		return from
	}
	t := from.UnixNano()
	return time.Unix(0, (t/step+1)*step).UTC()
}

// ComputeNextRunAt returns the next fire time after from for this schedule.
func (s Schedule) ComputeNextRunAt(from time.Time) (time.Time, error) {
	if strings.TrimSpace(s.Interval) != "" {
		d, err := ParseScheduleInterval(s.Interval)
		if err != nil {
			return time.Time{}, err
		}
		return nextIntervalBoundary(from, d), nil
	}
	if strings.TrimSpace(s.Cron) != "" {
		sched, err := cron.ParseStandard(strings.TrimSpace(s.Cron))
		if err != nil {
			return time.Time{}, err
		}
		return sched.Next(from), nil
	}
	return time.Time{}, fmt.Errorf("schedule has no interval or cron")
}
