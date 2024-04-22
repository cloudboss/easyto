package ctr2ami

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"golang.org/x/sys/unix"
)

const (
	FSTypeExt4 = "ext4"
	FSTypeXFS  = "xfs"
	FSTypeVFAT = "vfat"
)

const (
	blkidNoInfo      = 2
	execBits         = 0111
	fmtRaw           = "raw"
	fileMetadata     = "metadata.json"
	modeDirStd       = 0755
	partTypePrimary  = "primary"
	tarCodeTimestamp = 'Z'
	tenGigs          = 10737418240

	deviceDisk     = "/dev/sda"
	devicePartEFI  = "/dev/sda1"
	devicePartRoot = "/dev/sda2"

	dirCB         = "/__cb__"
	dirEFI        = "/boot"
	dirLibModules = "/lib/modules"
	dirMnt        = "/mnt"
	dirRoot       = "/"

	labelEFI  = "EFI"
	labelRoot = "ROOT"

	pathProcNetPNP = "/proc/net/pnp"
)

var (
	kernelArchiveRe = regexp.MustCompile(`^kernel-(.*)\.tar`)
)

type errExtract struct {
	code rune
}

// dd is a minimal implementation of the `dd` command, where `if` is always /dev/zero
// and `bs` is expected to be given in bytes.
func dd(of string, bs, count int) (err error) {
	dest, err := os.Create(of)
	if err != nil {
		return err
	}
	defer func() {
		destErr := dest.Close()
		if destErr != nil && err == nil {
			err = destErr
		}
	}()

	buf := bytes.NewBuffer(make([]byte, bs))
	rdr := bytes.NewReader(buf.Bytes())

	for i := 0; i < count; i++ {
		_, err = rdr.Seek(0, 0)
		if err != nil {
			return err
		}

		_, err = io.Copy(dest, rdr)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e errExtract) Error() string {
	msg := "unknown error"

	switch e.code {
	case tar.TypeBlock:
		msg = "unable to create block device"
	case tar.TypeChar:
		msg = "unable to create character device"
	case tar.TypeDir:
		msg = "unable to create directory"
	case tar.TypeFifo:
		msg = "unable to create fifo"
	case tar.TypeLink:
		msg = "unable to create hard link"
	case tar.TypeReg:
		msg = "unable to create file"
	case tar.TypeSymlink:
		msg = "unable to create symbolic link"
	case tarCodeTimestamp:
		msg = "unable to set timestamp"
	}

	return "error extracting tar archive: " + msg
}

func newErrExtract(code rune, wrap error) error {
	err := errExtract{code: code}
	return fmt.Errorf("%w: %w", err, wrap)
}

type Builder struct {
	BootloaderPath string
	CTRImageName   string
	FSType         string
	KernelPath     string
	PreinitPath    string
	VMImageDevice  string
	VMImageMount   string
	WorkingDir     string
	kernelVersion  string
}

type BuilderOpt func(*Builder)

func WithBootloaderPath(bootloaderPath string) BuilderOpt {
	return func(b *Builder) {
		b.BootloaderPath = bootloaderPath
	}
}

func WithCTRImageName(ctrImageName string) BuilderOpt {
	return func(b *Builder) {
		b.CTRImageName = ctrImageName
	}
}

func WithFSType(fstype string) BuilderOpt {
	return func(b *Builder) {
		b.FSType = fstype
	}
}

func WithKernelPath(kernelPath string) BuilderOpt {
	return func(b *Builder) {
		b.KernelPath = kernelPath
	}
}

func WithPreinitPath(preinitPath string) BuilderOpt {
	return func(b *Builder) {
		b.PreinitPath = preinitPath
	}
}

func WithVMImageDevice(vmImageDevice string) BuilderOpt {
	return func(b *Builder) {
		b.VMImageDevice = vmImageDevice
	}
}

func WithVMImageMount(vmImageMount string) BuilderOpt {
	return func(b *Builder) {
		b.VMImageMount = vmImageMount
	}
}

func WithWorkingDir(workingDir string) BuilderOpt {
	return func(b *Builder) {
		b.WorkingDir = workingDir
	}
}

func NewBuilder(opts ...BuilderOpt) (*Builder, error) {
	builder := &Builder{}
	for _, opt := range opts {
		opt(builder)
	}

	if len(builder.BootloaderPath) == 0 {
		return nil, errors.New("bootloader path must be defined")
	}

	if len(builder.KernelPath) == 0 {
		return nil, errors.New("kernel path must be defined")
	}

	kernelVersion, err := kernelVersionFromArchive(builder.KernelPath)
	if err != nil {
		return nil, err
	}
	builder.kernelVersion = kernelVersion

	return builder, nil
}

func (b *Builder) MakeVMImage() (err error) {
	ctrImageRef, err := name.ParseReference(b.CTRImageName)
	if err != nil {
		return fmt.Errorf("unable to parse container image name: %w", err)
	}
	fmt.Printf("ctr image ref: %s, fully qualified: %s\n", ctrImageRef, ctrImageRef.Name())

	ctrImage, err := remote.Image(ctrImageRef)
	if err != nil {
		return fmt.Errorf("unable to retrieve remote image: %w", err)
	}

	imageReader := mutate.Extract(ctrImage)
	defer imageReader.Close()

	err = untarReader(imageReader, dirMnt, false)
	if err != nil {
		return err
	}

	err = b.setupBootloader()
	if err != nil {
		return err
	}

	err = b.setupPreinit()
	if err != nil {
		return err
	}

	err = b.setupKernel()
	if err != nil {
		return err
	}

	err = b.setupResolver()
	if err != nil {
		return err
	}

	err = b.setupMetadata(ctrImage, filepath.Join(b.VMImageMount, dirCB, fileMetadata))
	if err != nil {
		return err
	}

	return nil
}

func kernelVersionFromArchive(kernelArchivePath string) (string, error) {
	kernelArchiveFile := filepath.Base(kernelArchivePath)
	result := kernelArchiveRe.FindStringSubmatch(kernelArchiveFile)
	if len(result) != 2 {
		return "", fmt.Errorf("unexpected format of kernel archive file: %s", kernelArchiveFile)
	}
	return result[1], nil
}

func (b *Builder) formatBootEntry(partUUID string) string {
	contentFmt := `linux /vmlinuz-%s
options rw root=PARTUUID=%s console=tty0 console=ttyS0,115200 earlyprintk=ttyS0,115200 consoleblank=0 ip=dhcp init=%s/preinit
`
	return fmt.Sprintf(contentFmt, b.kernelVersion, partUUID, dirCB)
}

func (b *Builder) setupBootloader() error {
	err := untarFile(b.BootloaderPath, b.VMImageMount, true)
	if err != nil {
		return err
	}

	devicePartRoot := b.VMImageDevice + "p2"
	cmd := exec.Command("blkid", "-s", "PARTUUID", "-o", "value", devicePartRoot)
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	partRootUUID := strings.TrimSpace(string(out))
	fmt.Printf("partition UUID: %s\n", partRootUUID)

	bootEntryPath := filepath.Join(b.VMImageMount, "boot/loader/entries/cb.conf")
	err = os.MkdirAll(filepath.Dir(bootEntryPath), 0755)
	if err != nil {
		return fmt.Errorf("unable to make directory %s: %w", bootEntryPath, err)
	}

	bootEntryContent := b.formatBootEntry(partRootUUID)

	bootEntry, err := os.Create(bootEntryPath)
	if err != nil {
		return fmt.Errorf("unable to create %s: %w", bootEntryPath, err)
	}

	_, err = io.WriteString(bootEntry, bootEntryContent)
	if err != nil {
		return fmt.Errorf("unable to write %s: %w", bootEntryPath, err)
	}

	return nil
}

func (b *Builder) setupPreinit() error {
	return untarFile(b.PreinitPath, b.VMImageMount, false)
}

func (b *Builder) setupInit() error {
	src, err := os.Open(b.PreinitPath)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", b.PreinitPath, err)
	}
	defer src.Close()

	destPath := filepath.Join(b.VMImageMount, dirCB, "init")
	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("unable to create %s: %w", destPath, err)
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		return fmt.Errorf("unable to copy %s to %s: %w", b.PreinitPath, destPath, err)
	}

	err = os.Chmod(destPath, 0755)
	if err != nil {
		return fmt.Errorf("unable to set permissions on %s: %w", destPath, err)
	}

	return nil
}

func (b *Builder) setupKernel() error {
	err := untarFile(b.KernelPath, b.VMImageMount, false)
	if err != nil {
		return err
	}

	libModulesPath := filepath.Join(b.VMImageMount, dirLibModules)
	if _, err = os.Stat(libModulesPath); os.IsNotExist(err) {
		err = os.MkdirAll(libModulesPath, modeDirStd)
		if err != nil {
			return fmt.Errorf("unable to create kernel modules directory: %w", err)
		}
	}

	linkPath := filepath.Join(dirLibModules, b.kernelVersion)
	linkTargetPath := filepath.Join(dirCB, dirLibModules, b.kernelVersion)
	err = os.Symlink(linkTargetPath, linkPath)
	if err != nil {
		return fmt.Errorf("unable to link %s to %s: %w", linkPath, linkTargetPath, err)
	}

	return nil
}

func (b *Builder) setupResolver() error {
	oldMask := syscall.Umask(0)
	defer syscall.Umask(oldMask)

	etcPath := filepath.Join(b.VMImageMount, "etc")
	if _, err := os.Stat(etcPath); os.IsNotExist(err) {
		err = os.MkdirAll(etcPath, 0755)
		if err != nil {
			return fmt.Errorf("unable to create directory %s: %w", etcPath, err)
		}
	}

	resolveConfPath := filepath.Join(etcPath, "resolv.conf")

	if _, err := os.Lstat(resolveConfPath); err == nil {
		err := os.Remove(resolveConfPath)
		if err != nil {
			return fmt.Errorf("unable to remove existing %s: %w", resolveConfPath, err)
		}
	} else if !(err == nil || os.IsNotExist(err)) {
		return fmt.Errorf("unable to stat %s: %w", resolveConfPath, err)
	}

	err := os.Symlink(pathProcNetPNP, resolveConfPath)
	if err != nil {
		return fmt.Errorf("unable to link %s to %s: %w", resolveConfPath,
			pathProcNetPNP, err)
	}

	return nil
}

func (b *Builder) setupMetadata(ctrImage v1.Image, metadataPath string) (err error) {
	var metadata *v1.ConfigFile

	metadata, err = ctrImage.ConfigFile()
	if err != nil {
		return fmt.Errorf("unable to get metadata from image: %w", err)
	}

	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("unable to create metadata file: %w", err)
	}
	defer func() {
		cfgErr := metadataFile.Close()
		if cfgErr != nil && err == nil {
			err = cfgErr
		}
	}()

	err = json.NewEncoder(metadataFile).Encode(metadata)
	if err != nil {
		return fmt.Errorf("unable to write metadata file: %w", err)
	}

	return nil
}

type ts struct {
	atime time.Time
	mtime time.Time
}

func untarReader(reader io.Reader, destDir string, verbose bool) error {
	oldMask := syscall.Umask(0)
	defer syscall.Umask(oldMask)

	timestamps := map[string]ts{}

	treader := tar.NewReader(reader)

	for {
		hdr, err := treader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		fi := hdr.FileInfo()
		dest := filepath.Join(destDir, hdr.Name)
		if verbose {
			fmt.Printf("untar: extracting %s\n", dest)
		}
		timestamps[dest] = ts{atime: hdr.AccessTime, mtime: hdr.ModTime}

		switch hdr.Typeflag {
		case tar.TypeBlock:
			dev := int(unix.Mkdev(uint32(hdr.Devmajor), uint32(hdr.Devminor)))
			err = syscall.Mknod(dest, syscall.S_IFBLK, dev)
			if err != nil {
				return newErrExtract(tar.TypeBlock, err)
			}
		case tar.TypeChar:
			dev := int(unix.Mkdev(uint32(hdr.Devmajor), uint32(hdr.Devminor)))
			err = syscall.Mknod(dest, syscall.S_IFCHR, dev)
			if err != nil {
				return newErrExtract(tar.TypeChar, err)
			}
		case tar.TypeDir:
			err = os.Mkdir(dest, fi.Mode())
			if err != nil && os.IsNotExist(err) {
				// Try to create directories individually with os.Mkdir so that permissions
				// match the archive, but if a subdirectory entry comes before the parent
				// directory, fall back to os.MkdirAll to create the hierarchy.
				err = os.MkdirAll(dest, fi.Mode())
				if err != nil {
					return newErrExtract(tar.TypeDir, err)
				}
			} else if err != nil && os.IsExist(err) {
				// Directory already exists, so just set the mode.
				err = os.Chmod(dest, fi.Mode())
				if err != nil {
					return newErrExtract(tar.TypeDir, err)
				}
			} else if err != nil {
				return newErrExtract(tar.TypeDir, err)
			}
		case tar.TypeFifo:
			err = syscall.Mkfifo(dest, uint32(hdr.Mode))
			if err != nil {
				return newErrExtract(tar.TypeFifo, err)
			}
		case tar.TypeLink:
			err = os.Link(filepath.Join(destDir, hdr.Linkname), dest)
			if err != nil {
				return newErrExtract(tar.TypeLink, err)
			}
		case tar.TypeReg:
			err = copyFile(treader, dest, fi.Mode())
			if err != nil {
				return newErrExtract(tar.TypeReg, err)
			}
		case tar.TypeSymlink:
			err = os.Symlink(hdr.Linkname, dest)
			if err != nil {
				return newErrExtract(tar.TypeSymlink, err)
			}
		}

		err = os.Lchown(dest, hdr.Uid, hdr.Gid)
		if err != nil {
			return err
		}
	}

	// Change timestamps at the end, otherwise creation of
	// entries within directories resets parent timestamps.
	for dest, timestamp := range timestamps {
		ats := unix.Timespec{
			Sec:  timestamp.atime.Unix(),
			Nsec: int64(timestamp.atime.Nanosecond()),
		}
		mts := unix.Timespec{
			Sec:  timestamp.mtime.Unix(),
			Nsec: int64(timestamp.mtime.Nanosecond()),
		}
		tss := []unix.Timespec{ats, mts}
		err := unix.UtimesNanoAt(unix.AT_FDCWD, dest, tss, unix.AT_SYMLINK_NOFOLLOW)
		if err != nil {
			return newErrExtract(tarCodeTimestamp, err)
		}
	}

	return nil
}

func copyFile(src io.Reader, dest string, perm os.FileMode) error {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, src)
	return err
}

func untarFile(srcFile, destDir string, verbose bool) error {
	reader, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("unable to open %s for reading: %w", srcFile, err)
	}
	defer reader.Close()

	err = untarReader(reader, destDir, verbose)
	if err != nil {
		return fmt.Errorf("unable to extract %s to %s: %w", srcFile, destDir, err)
	}

	return nil
}
