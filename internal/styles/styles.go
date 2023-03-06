// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package styles

import (
	"github.com/charmbracelet/lipgloss"
	"strconv"
)

var (
	_2Family        = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "39", Dark: "86"})
	_4Family        = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "208", Dark: "192"})
	_5Family        = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "203", Dark: "204"})
	DefaultStyle    = lipgloss.NewStyle().Foreground(lipgloss.NoColor{})
	NewVersionStyle = _2Family

	AppStyle           = lipgloss.NewStyle().Padding(1, 2)
	TitleStyle         = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#ECFD65"})
	StatusMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8E8E8E", Dark: "#747373"}).Render
	SecondaryTextStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})
)

func GetStyle(statusCode int) lipgloss.Style {
	statusString := strconv.Itoa(statusCode)
	switch statusCode / 100 {
	case 2:
		return _2Family.SetString(statusString)
	case 4:
		return _4Family.SetString(statusString)
	case 5:
		return _5Family.SetString(statusString)
	default:
		return DefaultStyle.SetString(statusString)
	}
}
