// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"io/fs"

	"nui/ui"
)

func uiDistFS() fs.FS {
	return ui.DistFS()
}
