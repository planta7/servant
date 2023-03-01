package main

import (
	"serve/cmd"
	"serve/internal"
)

var (
	version string
	commit  string
)

func main() {
	internal.SetBuildInfo(version, commit)
	cmd.Execute()
}
