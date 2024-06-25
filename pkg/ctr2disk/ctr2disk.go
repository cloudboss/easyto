package ctr2disk

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/cloudboss/easyto/pkg/login"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

const (
	devicePartRoot   = "/dev/sda2"
	modeDirStd       = 0755
	tarCodeMode      = 'Y'
	tarCodeTimestamp = 'Z'

	dirLibModules = "/lib/modules"
	dirMnt        = "/mnt"

	pathPrefixKernel = "./boot/vmlinuz-"
	pathProcNetPNP   = constants.DirProc + "/net/pnp"

	archiveBootloader = "boot.tar"
	archiveChrony     = "chrony.tar"
	archiveInit       = "init.tar"
	archiveKernel     = "kernel.tar"
	archiveSSH        = "ssh.tar"
)

var (
	fs = afero.NewOsFs()
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
	case tarCodeMode:
		msg = "unable to set permissions"
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
	AssetDir      string
	CTRImageName  string
	VMImageDevice string
	VMImageMount  string
	Services      []string
	LoginUser     string
	LoginShell    string
	Debug         bool

	kernelVersion  string
	pathBootloader string
	pathChrony     string
	pathInit       string
	pathKernel     string
	pathSSH        string
}

type BuilderOpt func(*Builder)

func WithAssetDir(assetDir string) BuilderOpt {
	return func(b *Builder) {
		b.AssetDir = assetDir
	}
}

func WithCTRImageName(ctrImageName string) BuilderOpt {
	return func(b *Builder) {
		b.CTRImageName = ctrImageName
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

func WithServices(services []string) BuilderOpt {
	return func(b *Builder) {
		b.Services = services
	}
}

func WithLoginUser(user string) BuilderOpt {
	return func(b *Builder) {
		b.LoginUser = user
	}
}

func WithLoginShell(loginShell string) BuilderOpt {
	return func(b *Builder) {
		b.LoginShell = loginShell
	}
}

func WithDebug(debug bool) BuilderOpt {
	return func(b *Builder) {
		b.Debug = debug
	}
}

func NewBuilder(opts ...BuilderOpt) (*Builder, error) {
	builder := &Builder{}
	for _, opt := range opts {
		opt(builder)
	}

	if len(builder.AssetDir) == 0 {
		return nil, errors.New("asset directory must be defined")
	}

	builder.pathBootloader = filepath.Join(builder.AssetDir, archiveBootloader)
	builder.pathChrony = filepath.Join(builder.AssetDir, archiveChrony)
	builder.pathKernel = filepath.Join(builder.AssetDir, archiveKernel)
	builder.pathInit = filepath.Join(builder.AssetDir, archiveInit)
	builder.pathSSH = filepath.Join(builder.AssetDir, archiveSSH)

	kernelVersion, err := kernelVersionFromArchive(builder.pathKernel)
	if err != nil {
		return nil, fmt.Errorf("unable to determine kernel version from archive: %w", err)
	}
	builder.kernelVersion = kernelVersion

	return builder, nil
}

func (b *Builder) MakeVMImage() (err error) {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	if b.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	ctrImageRef, err := name.ParseReference(b.CTRImageName)
	if err != nil {
		return fmt.Errorf("unable to parse container image name: %w", err)
	}
	slog.Debug("Container image reference", "short", ctrImageRef, "long", ctrImageRef.Name())

	ctrImage, err := remote.Image(ctrImageRef)
	if err != nil {
		return fmt.Errorf("unable to retrieve remote image: %w", err)
	}

	imageReader := mutate.Extract(ctrImage)
	defer imageReader.Close()

	err = untarReader(imageReader, dirMnt)
	if err != nil {
		return err
	}

	err = b.setupBootloader()
	if err != nil {
		return err
	}

	err = b.setupInit()
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

	err = b.setupServices()
	if err != nil {
		return err
	}

	err = b.setupMetadata(ctrImage, filepath.Join(b.VMImageMount, constants.DirETRoot,
		constants.FileMetadata))
	if err != nil {
		return err
	}

	return nil
}

func (b *Builder) formatBootEntry(partUUID string) string {
	contentFmt := `linux /vmlinuz-%s
options rw root=PARTUUID=%s console=tty0 console=ttyS0,115200 earlyprintk=ttyS0,115200 consoleblank=0 ip=dhcp init=%s/init
`
	return fmt.Sprintf(contentFmt, b.kernelVersion, partUUID, constants.DirETSbin)
}

func (b *Builder) setupBootloader() error {
	err := untarFile(b.pathBootloader, b.VMImageMount)
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
	slog.Debug("Partition UUID", "uuid", partRootUUID)

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

func (b *Builder) setupInit() error {
	return untarFile(b.pathInit, b.VMImageMount)
}

func (b *Builder) setupKernel() error {
	err := untarFile(b.pathKernel, b.VMImageMount)
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

	linkPath := filepath.Join(b.VMImageMount, dirLibModules, b.kernelVersion)
	linkTargetPath := filepath.Join(constants.DirETRoot, dirLibModules, b.kernelVersion)
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

func (b *Builder) setupServices() error {
	for _, svc := range b.Services {
		switch svc {
		case "chrony":
			err := b.setupChrony()
			if err != nil {
				return fmt.Errorf("unable to setup chrony: %w", err)
			}
		case "ssh":
			err := b.setupSSH()
			if err != nil {
				return fmt.Errorf("unable to setup ssh: %w", err)
			}
		default:
			return fmt.Errorf("unknown service: %s", b.Services)
		}
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

func (b *Builder) setupChrony() error {
	err := untarFile(b.pathChrony, b.VMImageMount)
	if err != nil {
		return err
	}

	_, _, err = login.AddSystemUser(fs, constants.ChronyUser, constants.ChronyUser,
		"/nonexistent", b.VMImageMount)
	if err != nil {
		return fmt.Errorf("unable to add chrony user: %w", err)
	}

	return nil
}

func (b *Builder) setupSSH() error {
	oldMask := syscall.Umask(0)
	defer syscall.Umask(oldMask)

	err := untarFile(b.pathSSH, b.VMImageMount)
	if err != nil {
		return err
	}

	_, _, err = login.AddSystemUser(fs, constants.SSHPrivsepUser, constants.SSHPrivsepUser,
		"/nonexistent", b.VMImageMount)
	if err != nil {
		return fmt.Errorf("unable to add ssh privsep user: %w", err)
	}

	dirSSHPrivsep := filepath.Join(b.VMImageMount, constants.SSHPrivsepDir)
	if err := fs.MkdirAll(dirSSHPrivsep, 0755); err != nil {
		return fmt.Errorf("unable to create %s: %w", dirSSHPrivsep, err)
	}
	if err := fs.Chmod(dirSSHPrivsep, 0711); err != nil {
		return fmt.Errorf("unable to set permissions on %s: %w", dirSSHPrivsep, err)
	}

	homeDir := filepath.Join(constants.DirETHome, b.LoginUser)
	_, _, err = login.AddLoginUser(fs, b.LoginUser, b.LoginUser, homeDir, b.LoginShell, b.VMImageMount)
	if err != nil {
		return fmt.Errorf("unable to add login user: %w", err)
	}

	return nil
}

func kernelVersionFromArchive(pathKernelArchive string) (string, error) {
	f, err := os.Open(pathKernelArchive)
	if err != nil {
		return "", fmt.Errorf("unable to open %s: %w", pathKernelArchive, err)
	}
	defer f.Close()

	pathKernelTarEntry := ""
	treader := tar.NewReader(f)

	for {
		hdr, err := treader.Next()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return "", fmt.Errorf("unable to read %s file entry: %w", pathKernelArchive, err)
		}
		if strings.HasPrefix(hdr.Name, pathPrefixKernel) {
			pathKernelTarEntry = hdr.Name
			break
		}
	}

	fields := strings.Split(pathKernelTarEntry, pathPrefixKernel)
	if len(fields) != 2 {
		return "", fmt.Errorf("unable to find kernel in %s", pathKernelArchive)
	}

	return fields[1], nil
}

func untarReader(reader io.Reader, destDir string) error {
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
		perm := fi.Mode()
		dest := filepath.Join(destDir, hdr.Name)
		slog.Debug("untar extracting", "dest", dest)
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
			err = os.Mkdir(dest, perm)
			if err != nil && os.IsNotExist(err) {
				// Try to create directories individually with os.Mkdir so that permissions
				// match the archive, but if a subdirectory entry comes before the parent
				// directory, fall back to os.MkdirAll to create the hierarchy.
				err = os.MkdirAll(dest, perm)
				if err != nil {
					return newErrExtract(tar.TypeDir, err)
				}
			} else if err != nil && os.IsExist(err) {
				// Directory already exists, so just set the mode.
				err = os.Chmod(dest, perm)
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
			err = copyFile(treader, dest, perm)
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

		// Lchown may unset setuid and setgid bits.
		if perm&os.ModeSetuid != 0 || perm&os.ModeSetgid != 0 {
			err = os.Chmod(dest, perm)
			if !(err == nil || os.IsNotExist(err)) {
				return newErrExtract(tarCodeMode, err)
			}
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

func untarFile(srcFile, destDir string) error {
	reader, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("unable to open %s for reading: %w", srcFile, err)
	}
	defer reader.Close()

	err = untarReader(reader, destDir)
	if err != nil {
		return fmt.Errorf("unable to extract %s to %s: %w", srcFile, destDir, err)
	}

	return nil
}
