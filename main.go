// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import "loop/cmd"

func main() {
	cmd.SetUIFS(uiDistFS)
	cmd.SetExtFS(extFilesFS)
	cmd.Execute()
}
