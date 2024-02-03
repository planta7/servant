// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package tui

import (
	"github.com/charmbracelet/lipgloss"
	"strconv"
)

var (
	Family2xx       = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "39", Dark: "86"})
	Family4xx       = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "208", Dark: "192"})
	Family5xx       = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "203", Dark: "204"})
	DefaultStyle    = lipgloss.NewStyle().Foreground(lipgloss.NoColor{})
	NewVersionStyle = Family2xx

	AppStyle           = lipgloss.NewStyle().Padding(1, 2)
	TitleStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#ECFD65"})
	StatusMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8E8E8E", Dark: "#747373"}).Render
	SecondaryTextStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})
)

func GetStyle(statusCode int) lipgloss.Style {
	statusString := strconv.Itoa(statusCode)
	switch statusCode / 100 {
	case 2:
		return Family2xx.SetString(statusString)
	case 4:
		return Family4xx.SetString(statusString)
	case 5:
		return Family5xx.SetString(statusString)
	default:
		return DefaultStyle.SetString(statusString)
	}
}
