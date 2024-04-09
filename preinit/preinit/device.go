package preinit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mvisonneau/go-ebsnvme/pkg/ebsnvme"
)

// linkEBSDevices creates symlinks for user-defined EBS device names such as /dev/sdf
// to the real underlying device if it differs. This is needed for NVME devices.
func linkEBSDevices(c chan error) {
	dirs, err := os.ReadDir("/sys/block")
	if err != nil {
		c <- fmt.Errorf("unable to get entries in /sys/block: %w", err)
		return
	}
	for _, dir := range dirs {
		deviceName := dir.Name()
		devicePath := filepath.Join("/dev", deviceName)
		deviceInfo, err := ebsnvme.ScanDevice(devicePath)

		if err != nil {
			// Skip any block devices that are not EBS volumes. The
			// ebsnvme.ScanDevice() function returns fmt.Errorf() errors rather
			// than custom error types, so check the contents of the strings
			// here. Any updates to the version of that dependency should ensure
			// that the error messages continue to return "AWS EBS" within them.
			if strings.Contains(err.Error(), "AWS EBS") {
				continue
			}
			c <- fmt.Errorf("unable to scan device %s: %w", devicePath, err)
			return
		}

		deviceLinkPath := deviceInfo.Name
		if !strings.HasPrefix(deviceLinkPath, "/") {
			deviceLinkPath = filepath.Join("/dev", deviceLinkPath)
		}

		if _, err := os.Stat(deviceLinkPath); os.IsNotExist(err) {
			err = os.Symlink(deviceName, deviceLinkPath)
			if err != nil {
				c <- fmt.Errorf("unable to link device %s to %s: %w",
					devicePath, deviceLinkPath, err)
				return
			}
		}
	}
	c <- nil
}

func deviceHasFS(blkidPath, devicePath string) (bool, error) {
	cmd := exec.Command(blkidPath, devicePath)
	err := cmd.Run()
	switch cmd.ProcessState.ExitCode() {
	case 0:
		return true, nil
	case 2:
		return false, nil
	default:
		return false, err
	}
}
