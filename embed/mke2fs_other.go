//go:build !linux || !amd64

package busybox

import "errors"

func Run(args ...string) error {
	return errors.New("busybox embedded runner not implemented for this architecture")
}
