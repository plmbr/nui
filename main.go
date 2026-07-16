// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import "loop/cmd"

func main() {
	cmd.SetVersion(version())
	cmd.SetUIFS(uiDistFS)
	cmd.Execute()
}
