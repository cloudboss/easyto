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
		devicePath := filepath.Join("/dev", dir.Name())
		device, err := ebsnvme.ScanDevice(devicePath)

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

		ebsDeviceName := device.Name
		if !strings.HasPrefix(ebsDeviceName, "/") {
			ebsDeviceName = filepath.Join("/dev", ebsDeviceName)
		}

		if _, err := os.Stat(ebsDeviceName); os.IsNotExist(err) {
			err = os.Symlink(devicePath, device.Name)
			if err != nil {
				c <- fmt.Errorf("unable to link device %s to %s: %w",
					device.Name, devicePath, err)
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
