package preinit

import (
	"errors"
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

		err = os.Symlink(deviceName, deviceLinkPath)
		if !(err == nil || os.IsExist(err)) {
			c <- fmt.Errorf("unable to create link %s: %w", deviceLinkPath, err)
			return
		}

		// Link partitions too if they exist.
		partitions, err := diskPartitions(deviceName)
		if err != nil {
			c <- fmt.Errorf("unable to get partitions for device %s: %w", deviceName, err)
			return
		}
		for _, partition := range partitions {
			partitionSuffix := partition.partition
			if deviceHasNumericSuffix(deviceInfo.Name) {
				partitionSuffix = "p" + partitionSuffix
			}
			partitionLinkPath := deviceLinkPath + partitionSuffix
			err = os.Symlink(partition.device, partitionLinkPath)
			if !(err == nil || os.IsExist(err)) {
				c <- fmt.Errorf("unable to create link %s: %w", partitionLinkPath, err)
				return
			}
		}
	}
	c <- nil
}

type partitionInfo struct {
	device    string
	partition string
}

func diskPartitions(device string) ([]partitionInfo, error) {
	partitions := []partitionInfo{}

	deviceDir := filepath.Join("/sys/block", device)
	entries, err := os.ReadDir(deviceDir)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory %s: %w", deviceDir, err)
	}

	for _, entry := range entries {
		deviceName := entry.Name()
		// Look for partition subdirectories, e.g. /sys/block/nvme0n1/nvme0n1*/.
		if entry.IsDir() && strings.HasPrefix(deviceName, device) {
			partitionFile := filepath.Join(deviceDir, deviceName, "partition")
			contents, err := os.ReadFile(partitionFile)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("unable to read file %s: %w", partitionFile, err)
			}
			partitions = append(partitions, partitionInfo{
				device:    deviceName,
				partition: strings.TrimSpace(string(contents)),
			})
		}
	}
	return partitions, nil
}

func deviceHasNumericSuffix(device string) bool {
	return len(device) > 0 && device[len(device)-1] >= '0' && device[len(device)-1] <= '9'
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
