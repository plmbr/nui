// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"time"

	"nui/internal/store"

	"github.com/spf13/cobra"
)

var (
	gcDryRun       bool
	gcJSON         bool
	gcUpdateMaxAge time.Duration
)

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Garbage-collect leftover nui state under ~/.nui and temp dirs",
	Long: `Removes orphaned files left by interrupted saves, deleted sessions, and
prior process lifetimes:

  - ~/.nui/*.tmp atomic-write leftovers
  - ~/.nui/sessions/<id> and workspaces/<id> not in data.json
  - unindexed or orphaned ~/.nui/runs/*.jsonl
  - stale $TMPDIR/nui-uploads/<id> and old $TMPDIR/nui-update-* dirs
  - stale agentSessions / sessionMessages keys in data.json

The server also runs this automatically on startup.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := store.GarbageCollect(store.GCOptions{
			DryRun:       gcDryRun,
			UpdateMaxAge: gcUpdateMaxAge,
		})
		if err != nil {
			return err
		}
		if gcJSON {
			data, err := store.MarshalGCResultJSON(result)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		if gcDryRun {
			fmt.Printf("dry-run: would free %s (%s)\n", formatGCBytes(result.BytesFreed), store.FormatGCResult(result))
			return nil
		}
		fmt.Printf("cleaned %s\n", store.FormatGCResult(result))
		return nil
	},
}

func formatGCBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func init() {
	gcCmd.Flags().BoolVar(&gcDryRun, "dry-run", false, "Report what would be removed without deleting")
	gcCmd.Flags().BoolVar(&gcJSON, "json", false, "Print result as JSON")
	gcCmd.Flags().DurationVar(&gcUpdateMaxAge, "update-max-age", 24*time.Hour, "Minimum age of $TMPDIR/nui-update-* dirs before removal")
	rootCmd.AddCommand(gcCmd)
}
