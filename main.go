package main

import (
	"fmt"
	"github.com/charmbracelet/log"
	"serve/cmd"
)

var (
	version string
	commit  string
)

func main() {
	log.Info(fmt.Sprintf("serve %s (%s)", version, commit))
	cmd.Execute()
}
