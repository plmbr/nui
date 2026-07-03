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

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage scheduled autonomous agent runs",
}

var scheduleListCmd = &cobra.Command{
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

var scheduleAddCmd = &cobra.Command{
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

var scheduleEnableCmd = &cobra.Command{
	Use:   "enable <id>",
	Short: "Enable a schedule",
	Args:  cobra.ExactArgs(1),
	RunE:  scheduleSetEnabled(true),
}

var scheduleDisableCmd = &cobra.Command{
	Use:   "disable <id>",
	Short: "Disable a schedule",
	Args:  cobra.ExactArgs(1),
	RunE:  scheduleSetEnabled(false),
}

var scheduleDeleteCmd = &cobra.Command{
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

var scheduleRunNowCmd = &cobra.Command{
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
	scheduleCmd.PersistentFlags().StringVar(&scheduleURL, "url", "", "Loop server base URL (default LOOP_URL or http://127.0.0.1:8080)")

	scheduleAddCmd.Flags().StringVarP(&scheduleAgentType, "agent-type", "a", "", "ADL agent id (must be promptMode:auto)")
	scheduleAddCmd.Flags().StringVar(&scheduleEvery, "every", "", "Fixed interval (e.g. 5m, 1h, 1d)")
	scheduleAddCmd.Flags().StringVar(&scheduleCron, "cron", "", "Cron expression (5-field)")
	scheduleAddCmd.Flags().StringVarP(&schedulePrompt, "prompt", "m", "", "Optional prompt override")
	scheduleAddCmd.Flags().StringVarP(&scheduleWorkingDir, "working-dir", "w", "", "Working directory for new sessions")
	scheduleAddCmd.Flags().StringVar(&scheduleName, "name", "", "Schedule display name")

	scheduleCmd.AddCommand(scheduleListCmd, scheduleAddCmd, scheduleEnableCmd, scheduleDisableCmd, scheduleDeleteCmd, scheduleRunNowCmd)
	rootCmd.AddCommand(scheduleCmd)
}
