// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"nui/internal/agent"
	"nui/internal/store"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage global environment variables",
}

var envListReveal bool

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured global environment variables",
	RunE: func(cmd *cobra.Command, args []string) error {
		secrets, err := store.LoadSecrets()
		if err != nil {
			return err
		}
		if secrets.Env == nil {
			secrets.Env = map[string]string{}
		}
		keys := make([]string, 0, len(secrets.Env))
		for key := range secrets.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			fmt.Println("No environment variables configured.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tTYPE\tVALUE")
		for _, key := range keys {
			value := secrets.Env[key]
			fmt.Fprintf(w, "%s\t%s\t%s\n", key, envVarType(key), formatEnvValue(key, value, envListReveal))
		}
		return w.Flush()
	},
}

var envGetReveal bool

var envGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "Print a global environment variable value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.TrimSpace(args[0])
		if key == "" {
			return fmt.Errorf("env key is required")
		}
		secrets, err := store.LoadSecrets()
		if err != nil {
			return err
		}
		value, ok := secrets.Env[key]
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("env var %q is not set", key)
		}
		fmt.Println(formatEnvValue(key, value, envGetReveal))
		return nil
	},
}

var envSetCmd = &cobra.Command{
	Use:   "set KEY=VALUE [KEY=VALUE...]",
	Short: "Set global environment variables in ~/.nui/secrets.json",
	Long: `Set one or more global environment variables.

Examples:
  nui env set MY_VAR=hello
  nui env set ANTHROPIC_API_KEY=sk-ant-...`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pairs, err := parseEnvPairs(args)
		if err != nil {
			return err
		}
		for _, pair := range pairs {
			if err := setUserSecret(pair.key, pair.value); err != nil {
				return err
			}
			if pair.value == "" {
				fmt.Fprintf(os.Stderr, "Unset %q\n", pair.key)
			} else {
				fmt.Fprintf(os.Stderr, "Set %q\n", pair.key)
			}
		}
		return nil
	},
}

var envUnsetCmd = &cobra.Command{
	Use:   "unset KEY [KEY...]",
	Short: "Remove global environment variables from ~/.nui/secrets.json",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, key := range args {
			key = strings.TrimSpace(key)
			if key == "" {
				return fmt.Errorf("env key is required")
			}
			if err := unsetUserSecret(key); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Unset %q\n", key)
		}
		return nil
	},
}

func init() {
	envListCmd.Flags().BoolVar(&envListReveal, "reveal", false, "show secret values unmasked")
	envGetCmd.Flags().BoolVar(&envGetReveal, "reveal", false, "show secret value unmasked")

	envCmd.AddCommand(envListCmd, envGetCmd, envSetCmd, envUnsetCmd)
	rootCmd.AddCommand(envCmd)
}

func parseEnvPair(arg string) (string, string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", "", fmt.Errorf("env assignment is required")
	}
	if key, value, ok := strings.Cut(arg, "="); ok {
		key = strings.TrimSpace(key)
		if key == "" {
			return "", "", fmt.Errorf("env key is required")
		}
		return key, strings.TrimSpace(value), nil
	}
	return "", "", fmt.Errorf("expected KEY=VALUE, got %q", arg)
}

func parseEnvPairs(args []string) ([]struct{ key, value string }, error) {
	out := make([]struct{ key, value string }, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			return nil, fmt.Errorf("env assignment is required")
		}
		if key, value, ok := strings.Cut(arg, "="); ok {
			key = strings.TrimSpace(key)
			if key == "" {
				return nil, fmt.Errorf("env key is required")
			}
			out = append(out, struct{ key, value string }{key, strings.TrimSpace(value)})
			continue
		}
		if i+1 >= len(args) {
			return nil, fmt.Errorf("expected KEY=VALUE or KEY VALUE, got %q", arg)
		}
		key := arg
		value := strings.TrimSpace(args[i+1])
		out = append(out, struct{ key, value string }{key, value})
		i++
	}
	return out, nil
}

func envVarType(key string) string {
	if agent.IsManagedCredentialKey(key) {
		return "managed"
	}
	return "custom"
}

func isSecretEnvKey(key string) bool {
	for _, spec := range agent.CredentialFieldSpecs() {
		if spec.Key == key {
			return spec.Secret
		}
	}
	return false
}

func formatEnvValue(key, value string, reveal bool) string {
	if reveal || !isSecretEnvKey(key) || value == "" {
		return value
	}
	return "********"
}

func setUserSecret(key, value string) error {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" {
		return fmt.Errorf("env key is required")
	}
	if store.IsReservedEnvKey(key) {
		return fmt.Errorf("reserved env key: %s", key)
	}
	current, err := store.LoadUserSecrets()
	if err != nil {
		return err
	}
	if current.Env == nil {
		current.Env = map[string]string{}
	}
	if value == "" {
		delete(current.Env, key)
	} else {
		current.Env[key] = value
	}
	if err := store.SaveSecrets(current); err != nil {
		return err
	}
	store.ApplyGlobalEnvToProcess()
	return nil
}

func unsetUserSecret(key string) error {
	return setUserSecret(key, "")
}
