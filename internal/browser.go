// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package internal

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

func LaunchBrowser(serverUrl string) error {
	_, err := url.ParseRequestURI(serverUrl)
	if err != nil {
		return err
	}
	if IsTestRun() { // :|
		return nil
	}
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", serverUrl).Start()
	case "darwin":
		err = exec.Command("open", serverUrl).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}
