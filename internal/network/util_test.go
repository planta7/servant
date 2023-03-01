// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package network

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLocalIP(t *testing.T) {
	tt := []struct {
		name     string
		expected error
	}{
		{
			"local ip",
			nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			res, err := LocalIP()
			assert.NotEmpty(t, res.String())
			assert.Equal(t, tc.expected, err)
		})
	}
}
