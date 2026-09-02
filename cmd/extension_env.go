// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"nui/internal/extensions"
	"nui/internal/store"

	"github.com/spf13/cobra"
)

var extensionEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage per-extension environment variables",
}

var extensionEnvListCmd = &cobra.Command{
	Use:   "list EXTENSION",
	Short: "List environment variables for an extension",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if err := requireExtension(name); err != nil {
			return err
		}
		env := store.ExtensionEnv(name)
		keys := store.ExtensionEnvKeys(name)
		if len(keys) == 0 {
			fmt.Printf("No environment variables configured for extension %q.\n", name)
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tVALUE")
		for _, key := range keys {
			fmt.Fprintf(w, "%s\t%s\n", key, env[key])
		}
		return w.Flush()
	},
}

var extensionEnvGetCmd = &cobra.Command{
	Use:   "get EXTENSION KEY",
	Short: "Print an extension environment variable value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		key := strings.TrimSpace(args[1])
		if err := requireExtension(name); err != nil {
			return err
		}
		env := store.ExtensionEnv(name)
		value, ok := env[key]
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("env var %q is not set for extension %q", key, name)
		}
		fmt.Println(value)
		return nil
	},
}

var extensionEnvSetCmd = &cobra.Command{
	Use:   "set EXTENSION KEY=VALUE [KEY=VALUE...]",
	Short: "Set environment variables for an extension",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if err := requireExtension(name); err != nil {
			return err
		}
		pairs, err := parseEnvPairs(args[1:])
		if err != nil {
			return err
		}
		for _, pair := range pairs {
			if store.IsReservedEnvKey(pair.key) {
				return fmt.Errorf("reserved env key: %s", pair.key)
			}
			if err := patchExtensionEnv(name, pair.key, pair.value); err != nil {
				return err
			}
			if pair.value == "" {
				fmt.Fprintf(os.Stderr, "Unset %q for extension %q\n", pair.key, name)
			} else {
				fmt.Fprintf(os.Stderr, "Set %q for extension %q\n", pair.key, name)
			}
		}
		return nil
	},
}

var extensionEnvUnsetCmd = &cobra.Command{
	Use:   "unset EXTENSION KEY [KEY...]",
	Short: "Remove environment variables for an extension",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if err := requireExtension(name); err != nil {
			return err
		}
		for _, key := range args[1:] {
			key = strings.TrimSpace(key)
			if key == "" {
				return fmt.Errorf("env key is required")
			}
			if err := patchExtensionEnv(name, key, ""); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Unset %q for extension %q\n", key, name)
		}
		return nil
	},
}

func init() {
	extensionEnvCmd.AddCommand(extensionEnvListCmd, extensionEnvGetCmd, extensionEnvSetCmd, extensionEnvUnsetCmd)
	extensionCmd.AddCommand(extensionEnvCmd)
}

func requireExtension(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("extension name is required")
	}
	entries, err := extensions.List()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name == name {
			return nil
		}
	}
	return fmt.Errorf("extension %q is not installed", name)
}

func patchExtensionEnv(name, key, value string) error {
	f, err := store.LoadUserExtensionEnv()
	if err != nil {
		return err
	}
	if f.Env == nil {
		f.Env = map[string]map[string]string{}
	}
	env := f.Env[name]
	if env == nil {
		env = map[string]string{}
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if value == "" {
		delete(env, key)
	} else {
		env[key] = value
	}
	f.Env[name] = env
	return store.SetExtensionEnv(name, env)
}
