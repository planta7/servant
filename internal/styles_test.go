package internal

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
			_2Family.SetString("200"),
		},
		{
			"not found",
			404,
			_4Family.SetString("404"),
		},
		{
			"internal server error",
			500,
			_5Family.SetString("500"),
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
