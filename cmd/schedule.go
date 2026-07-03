// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"loop/internal/loopclient"

	"github.com/spf13/cobra"
)

var (
	scheduleURL        string
	scheduleAgentType  string
	scheduleEvery      string
	scheduleCron       string
	schedulePrompt     string
	scheduleWorkingDir string
	scheduleName       string
)

// NewScheduleCmd returns the loop schedule command tree.
func NewScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage scheduled autonomous agent runs",
	}
	cmd.PersistentFlags().StringVar(&scheduleURL, "url", "", "Loop server base URL (default LOOP_URL or http://127.0.0.1:8080)")

	addCmd := newScheduleAddCmd()
	cmd.AddCommand(
		newScheduleListCmd(),
		addCmd,
		newScheduleEnableCmd(),
		newScheduleDisableCmd(),
		newScheduleDeleteCmd(),
		newScheduleRunNowCmd(),
	)
	return cmd
}

func newScheduleListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			client := loopclient.New(scheduleURL)
			items, err := client.ListSchedules(ctx)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Println("No schedules.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tAGENT\tWHEN\tENABLED\tNEXT")
			for _, s := range items {
				when := s.Interval
				if when == "" {
					when = s.Cron
				}
				enabled := "no"
				if s.Enabled {
					enabled = "yes"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", s.ID, s.Name, s.AgentType, when, enabled, s.NextRunAt)
			}
			return w.Flush()
		},
	}
}

func newScheduleAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a schedule",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			every := strings.TrimSpace(scheduleEvery)
			cronExpr := strings.TrimSpace(scheduleCron)
			if every == "" && cronExpr == "" {
				return fmt.Errorf("one of --every or --cron is required")
			}
			if every != "" && cronExpr != "" {
				return fmt.Errorf("--every and --cron are mutually exclusive")
			}
			agentType := strings.TrimSpace(scheduleAgentType)
			if agentType == "" {
				return fmt.Errorf("--agent-type is required")
			}
			name := strings.TrimSpace(scheduleName)
			if name == "" {
				name = agentType
			}
			client := loopclient.New(scheduleURL)
			s, err := client.CreateSchedule(ctx, loopclient.CreateScheduleRequest{
				Name:       name,
				AgentType:  agentType,
				Prompt:     strings.TrimSpace(schedulePrompt),
				WorkingDir: strings.TrimSpace(scheduleWorkingDir),
				Interval:   every,
				Cron:       cronExpr,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created schedule %s (%s)\n", s.ID, s.Name)
			return nil
		},
	}
	cmd.Flags().StringVarP(&scheduleAgentType, "agent-type", "a", "", "ADL agent id (must be promptMode:auto)")
	cmd.Flags().StringVar(&scheduleEvery, "every", "", "Fixed interval (e.g. 5m, 1h, 1d)")
	cmd.Flags().StringVar(&scheduleCron, "cron", "", "Cron expression (5-field)")
	cmd.Flags().StringVarP(&schedulePrompt, "prompt", "m", "", "Optional prompt override")
	cmd.Flags().StringVarP(&scheduleWorkingDir, "working-dir", "w", "", "Working directory for new sessions")
	cmd.Flags().StringVar(&scheduleName, "name", "", "Schedule display name")
	return cmd
}

func newScheduleEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <id>",
		Short: "Enable a schedule",
		Args:  cobra.ExactArgs(1),
		RunE:  scheduleSetEnabled(true),
	}
}

func newScheduleDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <id>",
		Short: "Disable a schedule",
		Args:  cobra.ExactArgs(1),
		RunE:  scheduleSetEnabled(false),
	}
}

func newScheduleDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			client := loopclient.New(scheduleURL)
			if err := client.DeleteSchedule(ctx, args[0]); err != nil {
				return err
			}
			fmt.Println("Deleted.")
			return nil
		},
	}
}

func newScheduleRunNowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run-now <id>",
		Short: "Trigger a schedule immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			client := loopclient.New(scheduleURL)
			s, err := client.RunScheduleNow(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Started schedule %s; last session %s\n", s.Name, s.LastSessionID)
			return nil
		},
	}
}

func scheduleSetEnabled(enabled bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		client := loopclient.New(scheduleURL)
		s, err := client.PatchSchedule(ctx, args[0], loopclient.PatchScheduleRequest{
			Enabled: &enabled,
		})
		if err != nil {
			return err
		}
		state := "disabled"
		if s.Enabled {
			state = "enabled"
		}
		fmt.Printf("Schedule %s is %s\n", s.Name, state)
		return nil
	}
}

func init() {
	rootCmd.AddCommand(NewScheduleCmd())
}
