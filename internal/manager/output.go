// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package manager

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/serve/internal/styles"
	"github.com/planta7/serve/internal/tui"
)

type OutputManager interface {
	Write(request *Request)
}

type logOutput struct {
}

func NewLogOutput() OutputManager {
	return &logOutput{}
}

func (l *logOutput) Write(request *Request) {
	statusText := styles.GetStyle(request.Status)
	contentLengthText := getContentLength(request.ContentLength)
	logLine := fmt.Sprintf("%s\t%v\t%s\t%s\t%s %s",
		request.RemoteAddress,
		request.Time,
		statusText,
		request.Method,
		request.Url,
		contentLengthText)
	log.Info(logLine)
}

type tuiOutput struct {
	model *tui.Model
}

func NewTuiOutput(model *tui.Model) OutputManager {
	return &tuiOutput{
		model: model,
	}
}

func (t *tuiOutput) Write(request *Request) {
	statusText := styles.GetStyle(request.Status)
	contentLengthText := getContentLength(request.ContentLength)
	remoteAddressPart := styles.SecondaryTextStyle.Render(fmt.Sprintf("from %s", request.RemoteAddress))
	title := fmt.Sprintf("%s %s %s", request.Method, request.Url, remoteAddressPart)

	contentPart := styles.SecondaryTextStyle.Render(fmt.Sprintf("%s %s", request.ContentType, contentLengthText))
	description := fmt.Sprintf("%s %v %s", statusText, request.Time, contentPart)
	t.model.Add(title, description)
}

func getContentLength(value uint64) string {
	if value != 0 {
		return fmt.Sprintf("(%d bytes)", value)
	}
	return ""
}
