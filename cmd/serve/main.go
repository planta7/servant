// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package main

import (
	"github.com/planta7/serve/cmd/serve/command"
	"github.com/planta7/serve/internal"
)

var (
	version string
	commit  string
)

func main() {
	internal.SetBuildInfo(version, commit)
	command.Execute()
	internal.CheckForUpdates(version)
}
