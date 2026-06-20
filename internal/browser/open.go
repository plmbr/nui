// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Open launches url in the system default browser.
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		return fmt.Errorf("opening browser is not supported on %s", runtime.GOOS)
	}
	return cmd.Start()
}
