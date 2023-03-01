// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package internal

import (
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestLaunchBrowser(t *testing.T) {
	tt := []struct {
		name     string
		url      string
		expected error
	}{
		{
			"valid url",
			"http://localhost:8080",
			nil,
		},
		{
			"not valid url",
			"http//localhost:8080",
			&url.Error{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			res := LaunchBrowser(tc.url)
			assert.IsType(t, tc.expected, res)
		})
	}
}
