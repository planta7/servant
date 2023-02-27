package main

import (
	"github.com/charmbracelet/log"
	"serve/cmd"
)

func main() {
	log.Info("serve v0.1") // TODO: get version
	cmd.Execute()
}
