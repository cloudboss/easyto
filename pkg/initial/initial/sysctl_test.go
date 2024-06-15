package initial

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_keyToPath(t *testing.T) {
	testCases := []struct {
		key    string
		result string
	}{
		{
			key:    "",
			result: "/proc/sys",
		},
		{
			key:    "kernel.poweroff_cmd",
			result: "/proc/sys/kernel/poweroff_cmd",
		},
		{
			key:    "net.netfilter.nf_log.0",
			result: "/proc/sys/net/netfilter/nf_log/0",
		},
	}
	for _, tc := range testCases {
		actual := keyToPath(tc.key)
		assert.Equal(t, tc.result, actual)
	}
}
