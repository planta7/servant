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
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", serverUrl).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", serverUrl).Start()
	case "darwin":
		err = exec.Command("open", serverUrl).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}
