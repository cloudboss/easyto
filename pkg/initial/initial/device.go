package initial

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/cloudboss/easyto/pkg/constants"
	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/mvisonneau/go-ebsnvme/pkg/ebsnvme"
	"golang.org/x/sys/unix"
)

// linkEBSDevices creates symlinks for user-defined EBS device names such as /dev/sdf
// to the real underlying device if it differs. This is needed for NVME devices.
func linkEBSDevices() error {
	dirs, err := os.ReadDir("/sys/block")
	if err != nil {
		return fmt.Errorf("unable to get entries in /sys/block: %w", err)
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
			return fmt.Errorf("unable to scan device %s: %w", devicePath, err)
		}

		deviceLinkPath := deviceInfo.Name
		if !strings.HasPrefix(deviceLinkPath, "/") {
			deviceLinkPath = filepath.Join("/dev", deviceLinkPath)
		}

		err = os.Symlink(deviceName, deviceLinkPath)
		if !(err == nil || os.IsExist(err)) {
			return fmt.Errorf("unable to create link %s: %w", deviceLinkPath, err)
		}

		// Link partitions too if they exist.
		partitions, err := diskPartitions(deviceName)
		if err != nil {
			return fmt.Errorf("unable to get partitions for device %s: %w", deviceName, err)
		}
		for _, partition := range partitions {
			partitionSuffix := partition.partition
			if deviceHasNumericSuffix(deviceInfo.Name) {
				partitionSuffix = "p" + partitionSuffix
			}
			partitionLinkPath := deviceLinkPath + partitionSuffix
			err = os.Symlink(partition.device, partitionLinkPath)
			if !(err == nil || os.IsExist(err)) {
				return fmt.Errorf("unable to create link %s: %w", partitionLinkPath, err)
			}
		}
	}
	return nil
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

func resizeRootVolume() error {
	rootDisk, rootPartition, err := findRootDevice()
	if err != nil {
		return fmt.Errorf("unable to find root device: %w", err)
	}

	err = resizeRootPartition(rootDisk, rootPartition)
	if err != nil {
		return err
	}

	return growFilesystem(rootPartition)
}

// findRootDevice returns the disk device and partition device for the root partition.
func findRootDevice() (string, string, error) {
	blkidPath := filepath.Join(constants.DirETSbin, "blkid")

	cmd := exec.Command(blkidPath, "-t", "PARTLABEL=root", "-o", "device")
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("unable to find partition with root label: %w", err)
	}

	rootPartition := strings.TrimSpace(string(out))
	dir, rootPartitionFile := filepath.Split(rootPartition)
	if dir != "/dev/" {
		return "", "", fmt.Errorf("unexpected blkid output trying to find root partition: %s", rootPartition)
	}

	blockEntries, err := os.ReadDir("/sys/block")
	if err != nil {
		return "", "", fmt.Errorf("unable to read /sys/block: %w", err)
	}

	for _, entry := range blockEntries {
		entryName := entry.Name()
		statPath := filepath.Join("/sys/block", entryName, rootPartitionFile)
		_, err := os.Stat(statPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", "", fmt.Errorf("unable to stat %s: %w", rootPartitionFile, err)
		}
		rootDisk := filepath.Join("/dev", entryName)
		return rootDisk, rootPartition, nil
	}

	return "", "", fmt.Errorf("unable to find root device")
}

func resizeRootPartition(rootDiskDevice, rootPartitionDevice string) error {
	disk, err := diskfs.Open(rootDiskDevice, diskfs.WithOpenMode(diskfs.ReadWrite))
	if err != nil {
		return fmt.Errorf("unable to open device %s: %w", rootDiskDevice, err)
	}

	table, err := disk.GetPartitionTable()
	if err != nil {
		return fmt.Errorf("unable to get partition table for device %s: %w", rootDiskDevice, err)
	}

	gptTable, ok := table.(*gpt.Table)
	if !ok {
		return fmt.Errorf("device %s does not have a GPT partition table", rootDiskDevice)
	}

	const expectedPartitions = 2

	// The image should have an EFI boot partition and a root partition.
	if len(gptTable.Partitions) != expectedPartitions {
		return fmt.Errorf("expected %d partitions, got %d", expectedPartitions, len(gptTable.Partitions))
	}

	// The last partition should be the root partition.
	rootPartition := gptTable.Partitions[len(gptTable.Partitions)-1]
	if rootPartition.Name != "root" {
		return fmt.Errorf("expected a partition named 'root', got '%s'", rootPartition.Name)
	}

	const gptHeaderSectors = 1
	const gptPartitionEntrySectors = 32
	const gptSectors = gptHeaderSectors + gptPartitionEntrySectors
	lastDataSector := disk.Size/int64(disk.LogicalBlocksize) - gptSectors - 1

	if int64(rootPartition.End) < lastDataSector {
		slog.Info("extending root partition", "last-partition-sector", rootPartition.End,
			"last-available-sector", lastDataSector)

		rootPartition.End = uint64(lastDataSector)
		rootPartition.Size = (rootPartition.End - rootPartition.Start + 1) * uint64(disk.LogicalBlocksize)

		// The Repair method resets the locations of the secondary GPT
		// header and partition entries to the end of the disk.
		err = gptTable.Repair(uint64(disk.Size))
		if err != nil {
			return fmt.Errorf("unable to reset end of partition table: %w", err)
		}

		// Rewrite the GPT table on disk.
		err = disk.Partition(gptTable)
		if err != nil {
			// The diskfs library uses the BLKRRPART ioctl to re-read the
			// partition table during a call to Partition, but if the disk
			// is mounted, it fails because the device is busy. When that
			// error is returned, ignore it. We'll call rereadPartition()
			// that uses the BLKPG ioctl instead.
			if !strings.Contains(err.Error(), "device or resource busy") {
				return fmt.Errorf("unable to resize root partition: %w", err)
			}
		}

		err = rereadPartition(disk, rootPartition, rootPartitionDevice, expectedPartitions)
		if err != nil {
			return fmt.Errorf("unable to re-read partition after resizing: %w", err)
		}

		slog.Info("root partition extended")
	}

	return nil
}

// Use the BLKPG ioctl to re-read a partition after resizing.
func rereadPartition(disk *disk.Disk, partition *gpt.Partition, devicePath string, num int) error {
	const blkpgNameLen = 64

	volname := [blkpgNameLen]uint8{}
	for i, b := range []byte(partition.Name) {
		volname[i] = uint8(b)
	}

	devname := [blkpgNameLen]uint8{}
	for i, b := range []byte(devicePath) {
		devname[i] = uint8(b)
	}

	bp := unix.BlkpgPartition{
		Start:   int64(partition.Start) * disk.LogicalBlocksize,
		Length:  int64(partition.Size),
		Pno:     int32(num),
		Devname: devname,
		Volname: volname,
	}

	arg := unix.BlkpgIoctlArg{
		Op:      unix.BLKPG_RESIZE_PARTITION,
		Datalen: int32(unsafe.Sizeof(unix.BlkpgPartition{})),
		Data:    (*byte)(unsafe.Pointer(&bp)),
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(disk.File.Fd()), uintptr(unix.BLKPG),
		uintptr(unsafe.Pointer(&arg)))
	if errno != 0 {
		return syscall.Errno(errno)
	}

	return nil
}

func growFilesystem(devicePath string) error {
	resize2fsPath := filepath.Join(constants.DirETSbin, "resize2fs")
	cmd := exec.Command(resize2fsPath, devicePath)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("unable to resize filesystem: %w", err)
	}
	return nil
}
