// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"testing"
	"time"
)

func TestValidateScheduleInput(t *testing.T) {
	auto := func(agentType string) bool { return agentType == "auto-agent" }

	tests := []struct {
		name    string
		s       Schedule
		wantErr bool
	}{
		{
			name: "valid interval",
			s: Schedule{Name: "Daily", AgentType: "auto-agent", Interval: "1h"},
		},
		{
			name: "valid cron",
			s: Schedule{Name: "Morning", AgentType: "auto-agent", Cron: "0 9 * * *"},
		},
		{
			name:    "missing name",
			s:       Schedule{AgentType: "auto-agent", Interval: "5m"},
			wantErr: true,
		},
		{
			name:    "both interval and cron",
			s:       Schedule{Name: "x", AgentType: "auto-agent", Interval: "5m", Cron: "0 9 * * *"},
			wantErr: true,
		},
		{
			name:    "neither interval nor cron",
			s:       Schedule{Name: "x", AgentType: "auto-agent"},
			wantErr: true,
		},
		{
			name: "valid runAt",
			s: Schedule{
				Name:      "Later",
				AgentType: "auto-agent",
				RunAt:     time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			},
		},
		{
			name:    "all three timing modes",
			s:       Schedule{Name: "x", AgentType: "auto-agent", Interval: "5m", Cron: "0 9 * * *", RunAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339)},
			wantErr: true,
		},
		{
			name:    "interval too short",
			s:       Schedule{Name: "x", AgentType: "auto-agent", Interval: "5s"},
			wantErr: true,
		},
		{
			name:    "non auto agent",
			s:       Schedule{Name: "x", AgentType: "manual", Interval: "5m"},
			wantErr: true,
		},
		{
			name:    "invalid cron",
			s:       Schedule{Name: "x", AgentType: "auto-agent", Cron: "not cron"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScheduleInput(tt.s, auto)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateScheduleInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseScheduleInterval(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"5m", 5 * time.Minute},
		{"1h", time.Hour},
		{"1d", 24 * time.Hour},
		{"2d", 48 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
	}
	for _, tt := range tests {
		d, err := ParseScheduleInterval(tt.in)
		if err != nil {
			t.Fatalf("ParseScheduleInterval(%q): %v", tt.in, err)
		}
		if d != tt.want {
			t.Fatalf("ParseScheduleInterval(%q) = %v, want %v", tt.in, d, tt.want)
		}
	}
}

func TestScheduleComputeNextRunAt(t *testing.T) {
	from := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)

	intervalSched := Schedule{Interval: "1h"}
	next, err := intervalSched.ComputeNextRunAt(from)
	if err != nil {
		t.Fatal(err)
	}
	wantHour := time.Date(2026, 7, 2, 11, 0, 0, 0, time.UTC)
	if !next.Equal(wantHour) {
		t.Fatalf("interval next = %v, want %v", next, wantHour)
	}

	// 11:03 + 5m interval aligns to 11:05, not 11:08.
	fiveMinFrom := time.Date(2026, 7, 2, 11, 3, 0, 0, time.UTC)
	fiveMinSched := Schedule{Interval: "5m"}
	next, err = fiveMinSched.ComputeNextRunAt(fiveMinFrom)
	if err != nil {
		t.Fatal(err)
	}
	wantFiveMin := time.Date(2026, 7, 2, 11, 5, 0, 0, time.UTC)
	if !next.Equal(wantFiveMin) {
		t.Fatalf("5m next = %v, want %v", next, wantFiveMin)
	}

	daySched := Schedule{Interval: "1d"}
	next, err = daySched.ComputeNextRunAt(from)
	if err != nil {
		t.Fatal(err)
	}
	wantDay := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	if !next.Equal(wantDay) {
		t.Fatalf("1d next = %v, want %v", next, wantDay)
	}

	cronSched := Schedule{Cron: "0 9 * * *"}
	next, err = cronSched.ComputeNextRunAt(from)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 7, 3, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("cron next = %v, want %v", next, want)
	}
}

func TestNextIntervalBoundary(t *testing.T) {
	tests := []struct {
		name     string
		from     time.Time
		interval time.Duration
		want     time.Time
	}{
		{
			name:     "5m from 11:03",
			from:     time.Date(2026, 7, 2, 11, 3, 0, 0, time.UTC),
			interval: 5 * time.Minute,
			want:     time.Date(2026, 7, 2, 11, 5, 0, 0, time.UTC),
		},
		{
			name:     "5m exactly on boundary advances",
			from:     time.Date(2026, 7, 2, 11, 5, 0, 0, time.UTC),
			interval: 5 * time.Minute,
			want:     time.Date(2026, 7, 2, 11, 10, 0, 0, time.UTC),
		},
		{
			name:     "1h from top of hour",
			from:     time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
			interval: time.Hour,
			want:     time.Date(2026, 7, 2, 11, 0, 0, 0, time.UTC),
		},
		{
			name:     "1d from mid day",
			from:     time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
			interval: 24 * time.Hour,
			want:     time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextIntervalBoundary(tt.from, tt.interval)
			if !got.Equal(tt.want) {
				t.Fatalf("nextIntervalBoundary() = %v, want %v", got, tt.want)
			}
		})
	}
}
