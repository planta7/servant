// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		waitForActivity(m.channel))
}
