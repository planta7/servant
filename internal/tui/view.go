// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package tui

func (m Model) View() string {
	return AppStyle.Render(m.list.View())
}
