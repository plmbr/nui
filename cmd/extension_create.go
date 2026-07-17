// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var extensionCreateCmd = &cobra.Command{
	Use:   "create [id]",
	Short: "Scaffold a programmatic extension package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		lang, _ := cmd.Flags().GetString("lang")
		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			dir = id
		}
		return scaffoldExtension(id, lang, dir)
	},
}

func init() {
	extensionCreateCmd.Flags().String("lang", "python", "language: python, npm, go")
	extensionCreateCmd.Flags().String("dir", "", "output directory (default: extension id)")
	extensionCmd.AddCommand(extensionCreateCmd)
}

func scaffoldExtension(id, lang, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	switch strings.ToLower(lang) {
	case "python", "pip":
		return scaffoldPythonExtension(id, dir)
	case "npm", "ts", "typescript":
		return scaffoldNPMExtension(id, dir)
	case "go":
		return scaffoldGoExtension(id, dir)
	default:
		return fmt.Errorf("unsupported lang %q", lang)
	}
}

func scaffoldPythonExtension(id, dir string) error {
	pyproject := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"

[project.scripts]
nui-ext = "host:main"

[tool.nui]
id = "%s"
displayName = "%s"
`, id, id, id)
	host := `import sys
sys.path.insert(0, "../../harness-sdk")
from nui_extension import NuiExtension

class MyExtension(NuiExtension):
    def get_harnesses(self):
        return [{"id": "echo", "displayName": "Echo"}]

    def run_harness(self, harness_id, message, ctx=None):
        if harness_id == "echo":
            yield message

def main():
    MyExtension().serve()
`
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "host.py"), []byte(host), 0755)
}

func scaffoldNPMExtension(id, dir string) error {
	pkg := fmt.Sprintf(`{
  "name": "@local/%s",
  "version": "0.1.0",
  "type": "module",
  "bin": { "nui-ext": "host.js" },
  "nui": { "id": "%s", "displayName": "%s" }
}
`, id, id, id)
	host := `import { NuiExtension } from "../../sdk/typescript/NuiExtension.ts";

class MyExtension extends NuiExtension {
  getHarnesses() {
    return [{ id: "echo", displayName: "Echo" }];
  }
  async *runHarness(harnessId, message) {
    if (harnessId === "echo") yield message;
  }
}

new MyExtension().serve();
`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "host.js"), []byte(host), 0755)
}

func scaffoldGoExtension(id, dir string) error {
	mod := "module " + id + "\n\ngo 1.22\n\nrequire nui/sdk/go/nuiextension v0.0.0\n\nreplace nui/sdk/go/nuiextension => ../../sdk/go/nuiextension\n"
	main := `package main

import (
	"context"
	"nui/sdk/go/nuiextension"
)

type ext struct{ nuiextension.Base }

func (e *ext) GetHarnesses() []map[string]any {
	return []map[string]any{{"id": "echo", "displayName": "Echo"}}
}

func (e *ext) RunHarness(ctx context.Context, harnessID, message string, params map[string]any, emit func(map[string]any)) error {
	if harnessID == "echo" {
		emit(map[string]any{"type": "text", "content": message})
	}
	return nil
}

func main() {
	nuiextension.ServeStdio(&ext{})
}
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(main), 0644)
}
