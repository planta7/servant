package internal

import (
	"github.com/charmbracelet/lipgloss"
	"strconv"
)

var (
	_2Family = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "39",
			Dark:  "86",
		})
	_4Family = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "208",
			Dark:  "192",
		})
	_5Family = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "203",
			Dark:  "204",
		})
	DefaultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{})
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
