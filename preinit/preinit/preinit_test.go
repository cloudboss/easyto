package preinit

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getenv(t *testing.T) {
	testCases := []struct {
		env      []string
		envVar   string
		expected string
	}{
		{
			env:      []string{},
			envVar:   "",
			expected: "",
		},
		{
			env: []string{
				"HOME=/root",
				"PATH=/bin:/sbin",
			},
			envVar:   "PATH",
			expected: "/bin:/sbin",
		},
	}
	for _, tc := range testCases {
		ev := getenv(tc.env, tc.envVar)
		assert.Equal(t, tc.expected, ev)
	}
}

func Test_parseMode(t *testing.T) {
	testCases := []struct {
		mode   string
		result fs.FileMode
		err    error
	}{
		{
			mode:   "",
			result: 0755,
			err:    nil,
		},
		{
			mode:   "abc",
			result: 0,
			err:    errors.New("invalid mode abc"),
		},
		{
			mode:   "-1",
			result: 0,
			err:    errors.New("invalid mode -1"),
		},
		{
			mode:   "256",
			result: 0,
			err:    errors.New("invalid mode 256"),
		},
		{
			mode:   "1234567890",
			result: 0,
			err:    errors.New("invalid mode 1234567890"),
		},
	}
	for _, tc := range testCases {
		actual, err := parseMode(tc.mode)
		fmt.Printf("mode: %s\n", actual)
		assert.Equal(t, tc.result, actual)
		assert.Equal(t, tc.err, err)
	}
}
