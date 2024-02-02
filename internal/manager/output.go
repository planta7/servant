// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package manager

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/planta7/servant/internal"
	"github.com/planta7/servant/internal/tui"
	"os"
	"syscall"
)

type OutputManager interface {
	Init(path string, addresses []string)
	Write(request *Request)
}

type logOutput struct {
}

func NewLogOutput() OutputManager {
	return &logOutput{}
}

func (l *logOutput) Write(request *Request) {
	statusText := tui.GetStyle(request.Status)
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

func (l *logOutput) Init(path string, addresses []string) {
	log.Info(fmt.Sprintf("Serving %s at %s", path, addresses))
}

type tuiOutput struct {
	model tui.Model
}

func NewTuiOutput() OutputManager {
	return &tuiOutput{}
}

func (t *tuiOutput) Write(request *Request) {
	statusText := tui.GetStyle(request.Status)
	contentLengthText := getContentLength(request.ContentLength)
	remoteAddressPart := tui.SecondaryTextStyle.Render(fmt.Sprintf("from %s", request.RemoteAddress))
	title := fmt.Sprintf("%s %s %s", request.Method, request.Url, remoteAddressPart)

	contentPart := tui.SecondaryTextStyle.Render(fmt.Sprintf("%s %s", request.ContentType, contentLengthText))
	description := fmt.Sprintf("%s %v %s", statusText, request.Time, contentPart)
	t.model.Add(title, description)
}

func (t *tuiOutput) Init(path string, addresses []string) {
	servingInfo := fmt.Sprintf("servant %s (%s)\nServing %s at %s",
		internal.ServantInfo.Version,
		internal.ServantInfo.GetShortCommit(),
		path,
		addresses,
	)
	t.model = tui.NewModel(servingInfo)
	go func() {
		if _, err := tea.NewProgram(t.model).Run(); err != nil {
			log.Fatal("Error running TUI", "error", err.Error())
		}
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(syscall.SIGTERM)
	}()
}

func getContentLength(value uint64) string {
	if value != 0 {
		return fmt.Sprintf("(%d bytes)", value)
	}
	return ""
}
