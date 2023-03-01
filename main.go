// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package main

import (
	"github.com/planta7/serve/cmd"
	"github.com/planta7/serve/internal"
)

var (
	version string
	commit  string
)

func main() {
	internal.SetBuildInfo(version, commit)
	cmd.Execute()
	internal.CheckForUpdates(version)
}
