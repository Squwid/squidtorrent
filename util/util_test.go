package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBytes(t *testing.T) {
	tests := map[string]struct {
		bytes     int
		formatted string
	}{
		"size of debian": {
			bytes:     353370112,
			formatted: "353.4 MB",
		},
		"only bytes": {
			bytes:     124,
			formatted: "124 B",
		},
		"kilo": {
			bytes:     9284,
			formatted: "9.3 kB",
		},
		"gig": {
			bytes:     5235745682,
			formatted: "5.2 GB",
		},
	}

	for _, test := range tests {
		f := FormatBytes(test.bytes)
		assert.Equal(t, test.formatted, f)
	}
}
