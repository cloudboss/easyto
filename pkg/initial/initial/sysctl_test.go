package initial

import (
	"path/filepath"
	"testing"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/stretchr/testify/assert"
)

func Test_keyToPath(t *testing.T) {
	testCases := []struct {
		key    string
		result string
	}{
		{
			key:    "",
			result: filepath.Join(constants.DirProc, "sys"),
		},
		{
			key:    "kernel.poweroff_cmd",
			result: filepath.Join(constants.DirProc, "sys/kernel/poweroff_cmd"),
		},
		{
			key:    "net.netfilter.nf_log.0",
			result: filepath.Join(constants.DirProc, "sys/net/netfilter/nf_log/0"),
		},
	}
	for _, tc := range testCases {
		actual := keyToPath(tc.key)
		assert.Equal(t, tc.result, actual)
	}
}
