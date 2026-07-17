// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"nui/internal/memory"
	"nui/internal/store"

	"github.com/spf13/cobra"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage nui persistent memory files",
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List memory files",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings, err := store.LoadSettings()
		if err != nil {
			return err
		}
		summary, err := memory.ListSummary(settings)
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "SCOPE\tID\tBYTES\tMODE\n")
		fmt.Fprintf(w, "user\tuser\t%d\t%s\n", summary.User.Size, summary.User.Mode)
		for _, entry := range summary.Agents {
			fmt.Fprintf(w, "agent\t%s\t%d\t%s\n", entry.AgentID, entry.Size, entry.Mode)
		}
		return w.Flush()
	},
}

var memoryShowCmd = &cobra.Command{
	Use:   "show [user|agent-id]",
	Short: "Show memory content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := strings.TrimSpace(args[0])
		var content string
		var err error
		if target == "user" {
			content, err = memory.ReadUser()
		} else {
			content, err = memory.ReadAgent(target)
		}
		if err != nil {
			return err
		}
		fmt.Print(content)
		if content != "" && !strings.HasSuffix(content, "\n") {
			fmt.Println()
		}
		return nil
	},
}

var memoryEditCmd = &cobra.Command{
	Use:   "edit [user|agent-id]",
	Short: "Edit memory in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := strings.TrimSpace(args[0])
		var path string
		var err error
		if target == "user" {
			path, err = memory.UserPath()
		} else {
			path, err = memory.AgentPath(target)
		}
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte{}, 0644); err != nil {
				return err
			}
		}
		editor := strings.TrimSpace(os.Getenv("EDITOR"))
		if editor == "" {
			if runtime.GOOS == "windows" {
				editor = "notepad"
			} else {
				editor = "vi"
			}
		}
		parts := strings.Fields(editor)
		if len(parts) == 0 {
			return fmt.Errorf("EDITOR is empty")
		}
		editCmd := exec.Command(parts[0], append(parts[1:], path)...)
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr
		return editCmd.Run()
	},
}

func init() {
	memoryCmd.AddCommand(memoryListCmd, memoryShowCmd, memoryEditCmd)
	rootCmd.AddCommand(memoryCmd)
}
