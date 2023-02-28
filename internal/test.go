package internal

import (
	"flag"
	"testing"
)

func init() {
	testing.Init()
	flag.Parse()
}

func IsTestRun() bool {
	return flag.Lookup("test.v").Value.(flag.Getter).Get().(bool)
}
