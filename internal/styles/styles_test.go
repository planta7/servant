// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetStyle(t *testing.T) {
	tt := []struct {
		name       string
		statusCode int
		expected   lipgloss.Style
	}{
		{
			"ok",
			200,
			Family2xx.SetString("200"),
		},
		{
			"not found",
			404,
			Family4xx.SetString("404"),
		},
		{
			"internal server error",
			500,
			Family5xx.SetString("500"),
		},
		{
			"redirection",
			301,
			DefaultStyle.SetString("301"),
		},
		{
			"switching protocols",
			101,
			DefaultStyle.SetString("101"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			res := GetStyle(tc.statusCode)
			assert.Equal(t, tc.expected, res)
		})
	}
}
