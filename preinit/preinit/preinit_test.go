package cbinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_findExecutableInPath(t *testing.T) {
	testCases := []struct {
		path       string
		executable string
		result     string
		err        error
	}{
		{
			path:       "/bin:/sbin:/usr/local/bin",
			executable: "bash",
			result:     "/bin/bash",
			err:        nil,
		},
		{
			path:       "~/bin:/bin:/sbin:/usr/local/bin",
			executable: "crictl",
			result:     "/usr/local/bin/crictl",
			err:        nil,
		},
		{
			path:       "~/bin:/bin:/sbin:/usr/local/bin",
			executable: "abcheyheyhey",
			result:     "",
			err:        executableNotFound,
		},
	}
	for _, tc := range testCases {
		result, err := findExecutableInPath(tc.executable, tc.path)
		assert.Equal(t, tc.result, result)
		assert.Equal(t, tc.err, err)
	}
}

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
