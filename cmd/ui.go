// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"embed"
	"fmt"
	"io/fs"

	"loop/internal/server"

	"github.com/spf13/cobra"
)

var port int
var uiFS func() fs.FS
var extFS func() embed.FS

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the web UI server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Starting web server on port %d...\n", port)
		return server.Start(port, uiFS(), extFS())
	},
}

func init() {
	uiCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(uiCmd)
}

// SetUIFS is called from main to inject the embedded UI FS provider.
func SetUIFS(fn func() fs.FS) {
	uiFS = fn
}

// SetExtFS is called from main to inject the embedded extensions FS provider.
func SetExtFS(fn func() embed.FS) {
	extFS = fn
}
