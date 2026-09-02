// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func confirm(cmd *cobra.Command, prompt string) (bool, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N] ", prompt)
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && !strings.Contains(err.Error(), "EOF") {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

func isInteractive(cmd *cobra.Command) bool {
	f, ok := cmd.InOrStdin().(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func confirmOverwrite(cmd *cobra.Command, yes bool, resource, name string) (bool, error) {
	if yes {
		return true, nil
	}
	if !isInteractive(cmd) {
		return false, fmt.Errorf("%s %q already exists (use -y to overwrite)", resource, name)
	}
	prompt := fmt.Sprintf("%s %q is already installed. Overwrite?", resourceLabel(resource), name)
	ok, err := confirm(cmd, prompt)
	if err != nil {
		return false, err
	}
	if !ok {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
	}
	return ok, nil
}

func confirmDirOverwrite(cmd *cobra.Command, yes bool, dir string) (bool, error) {
	if yes {
		return true, nil
	}
	if !isInteractive(cmd) {
		return false, fmt.Errorf("directory %q is not empty (use -y to overwrite)", dir)
	}
	ok, err := confirm(cmd, fmt.Sprintf("Directory %q is not empty. Overwrite?", dir))
	if err != nil {
		return false, err
	}
	if !ok {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
	}
	return ok, nil
}

func dirExistsNonEmpty(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%q is not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func installWithOverwrite(cmd *cobra.Command, yes bool, resource, name string, install func(overwrite bool) (string, error)) (string, error) {
	result, err := install(false)
	if err == nil {
		return result, nil
	}
	if !isAlreadyExists(err) {
		return "", err
	}
	existingName := alreadyExistsName(err, name)
	ok, err := confirmOverwrite(cmd, yes, resource, existingName)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("cancelled")
	}
	return install(true)
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "already exists")
}

func alreadyExistsName(err error, fallback string) string {
	msg := err.Error()
	const prefix = "already exists: "
	if idx := strings.Index(msg, prefix); idx >= 0 {
		name := strings.TrimSpace(msg[idx+len(prefix):])
		if name != "" {
			return strings.Trim(name, `"`)
		}
	}
	return fallback
}

func resourceLabel(resource string) string {
	switch resource {
	case "agent":
		return "Agent"
	case "extension":
		return "Extension"
	case "skill":
		return "Skill"
	default:
		return resource
	}
}
