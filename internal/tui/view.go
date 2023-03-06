// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package tui

import "github.com/planta7/serve/internal/styles"

func (m Model) View() string {
	return styles.AppStyle.Render(m.list.View())
}
