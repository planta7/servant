// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package internal

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/servant/internal/tui"
	"io"
	"net/http"
)

const (
	repoUrl     = "https://github.com/planta7/servant"
	releasesUrl = "https://api.github.com/repos/planta7/servant/releases/latest"
	tagKey      = "tag_name"
)

func CheckForUpdates(current string) {
	res, err := http.Get(releasesUrl)
	if err != nil {
		log.Warn("Error while querying for last release", "error", err)
		return
	}
	resBody, _ := io.ReadAll(res.Body)
	resMap := map[string]any{}
	if err = json.Unmarshal(resBody, &resMap); err != nil {
		log.Warn("Error while parsing response", "error", err)
		return
	}
	tagName := resMap[tagKey]
	if tagName == nil {
		log.Warn("Got nil while reading tag_name", "error", err)
		return
	}
	latest := tagName.(string)[1:]
	if current != latest {
		message := tui.NewVersionStyle.Render(
			fmt.Sprintf("\nThere is a new version available (v%s). Go to %s for more details.\n", latest, repoUrl))
		fmt.Println(message)
	}
}
