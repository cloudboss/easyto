package ctr2ami

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"libguestfs.org/guestfs"
)

const (
	FSTypeExt4 = "ext4"
	FSTypeXFS  = "xfs"
	FSTypeVFAT = "vfat"
)

const (
	execBits    = 0111
	fmtRaw      = "raw"
	blkidNoInfo = 2
	tenGigs     = 10737418240

	deviceDisk     = "/dev/sda"
	devicePartEFI  = "/dev/sda1"
	devicePartRoot = "/dev/sda2"

	dirCB         = "/__cb__"
	dirEFI        = "/boot"
	dirLibModules = "/lib/modules"
	dirRoot       = "/"

	fileMetadata = "metadata.json"

	labelEFI  = "EFI"
	labelRoot = "ROOT"

	modeDirStd = 0755

	partTypePrimary = "primary"
)

var (
	kernelArchiveRe = regexp.MustCompile(`^kernel-(.*)\.tar`)
)

type errExtract struct {
	fileType rune
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
	what := ""
	switch e.fileType {
	case tar.TypeBlock:
		what = "block device"
	case tar.TypeChar:
		what = "character device"
	case tar.TypeDir:
		what = "directory"
	case tar.TypeFifo:
		what = "fifo"
	case tar.TypeLink:
		what = "hard link"
	case tar.TypeReg:
		what = "file"
	case tar.TypeSymlink:
		what = "symbolic link"
	default:
		return "unknown file type while extracting archive"
	}
	format := "unable to create %s while extracting archive"
	return fmt.Sprintf(format, what)
}

func newErrExtract(fileType rune, wrap error) error {
	err := errExtract{fileType: fileType}
	return fmt.Errorf("%w: %w", err, wrap)
}

type Builder struct {
	BootloaderPath string
	CacheHome      string
	FSType         string
	Image          string
	InitPath       string
	KernelPath     string
	WorkingDir     string
	guestfs        *guestfs.Guestfs
	kernelVersion  string
}

type BuilderOpt func(*Builder)

func WithBootloaderPath(bootloaderPath string) BuilderOpt {
	return func(b *Builder) {
		b.BootloaderPath = bootloaderPath
	}
}

func WithCacheHome(dir string) BuilderOpt {
	return func(b *Builder) {
		b.CacheHome = dir
	}
}

func WithFSType(fstype string) BuilderOpt {
	return func(b *Builder) {
		b.FSType = fstype
	}
}

func WithImage(image string) BuilderOpt {
	return func(b *Builder) {
		b.Image = image
	}
}

func WithInitPath(initPath string) BuilderOpt {
	return func(b *Builder) {
		b.InitPath = initPath
	}
}

func WithKernelPath(kernelPath string) BuilderOpt {
	return func(b *Builder) {
		b.KernelPath = kernelPath
	}
}

func WithWorkingDir(workingDir string) BuilderOpt {
	return func(b *Builder) {
		b.WorkingDir = workingDir
	}
}

func NewBuilder(opts ...BuilderOpt) (*Builder, error) {
	g, err := guestfs.Create()
	if err != nil {
		return nil, err
	}
	builder := &Builder{guestfs: g}
	for _, opt := range opts {
		opt(builder)
	}

	if len(builder.BootloaderPath) == 0 {
		return nil, errors.New("bootloader path must be defined")
	}

	if len(builder.CacheHome) == 0 {
		return nil, errors.New("cache home must be defined")
	}

	if len(builder.KernelPath) == 0 {
		return nil, errors.New("kernel path must be defined")
	}

	err = os.MkdirAll(builder.CacheHome, modeDirStd)
	if err != nil {
		return nil, fmt.Errorf("unable to create cache home: %w", err)
	}

	builder.kernelVersion, err = kernelVersionFromArchive(builder.KernelPath)
	if err != nil {
		return nil, err
	}

	return builder, err
}

type cachedFiles struct {
	hasCacheDir      bool
	hasVMImageFile   bool
	hasRootFSArchive bool
	hasMetadata      bool
}

func (b *Builder) getCached(paths cachePaths) (cachedFiles, error) {
	cache := cachedFiles{}

	if _, err := os.Stat(paths.imageCache); os.IsNotExist(err) {
		return cache, nil
	} else if err != nil {
		return cache, err
	}
	cache.hasCacheDir = true

	if _, err := os.Stat(paths.vmImageFile); os.IsNotExist(err) {
		return cache, nil
	} else if err != nil {
		return cache, err
	}
	cache.hasVMImageFile = true

	if _, err := os.Stat(paths.rootFSArchive); os.IsNotExist(err) {
		return cache, nil
	} else if err != nil {
		return cache, err
	}
	cache.hasRootFSArchive = true

	if _, err := os.Stat(paths.metadata); os.IsNotExist(err) {
		return cache, nil
	} else if err != nil {
		return cache, err
	}
	cache.hasMetadata = true

	// TODO: compare checksums of files first.

	return cache, nil
}

func cleanCache(paths ...string) error {
	for _, file := range paths {
		err := os.Remove(file)
		if !(err == nil || os.IsNotExist(err)) {
			return fmt.Errorf("unable to remove cached file %s: %w", file, err)
		}
	}
	return nil
}

type cachePaths struct {
	imageCache    string
	metadata      string
	rootFSArchive string
	vmImageFile   string
}

func (b *Builder) MakeRawVMImage(ctrImageName string) (err error) {
	var (
		ctrImageDesc *remote.Descriptor
		ctrImageRef  name.Reference
		gfs          *guestfs.Guestfs
	)

	ctrImageRef, err = name.ParseReference(ctrImageName)
	if err != nil {
		return fmt.Errorf("unable to parse container image name: %w", err)
	}
	fmt.Printf("ctr image ref: %s, fully qualified: %s\n", ctrImageRef, ctrImageRef.Name())

	ctrImageDesc, err = remote.Get(ctrImageRef)
	if err != nil {
		return fmt.Errorf("unable to retrieve container image descriptor: %w", err)
	}

	imageCachePath := filepath.Join(b.CacheHome, ctrImageDesc.Digest.Hex)
	paths := cachePaths{
		imageCache:    imageCachePath,
		metadata:      filepath.Join(imageCachePath, fileMetadata),
		rootFSArchive: filepath.Join(imageCachePath, "rootfs.tar"),
		vmImageFile:   filepath.Join(imageCachePath, "vm.img"),
	}

	cache, err := b.getCached(paths)
	if err != nil {
		return fmt.Errorf("unable to check if VM image is cached: %w", err)
	}

	if cache.hasVMImageFile && cache.hasRootFSArchive && cache.hasMetadata {
		return nil
	} else {
		// Clear out the cache if it is in a partial state.
		err = cleanCache(paths.vmImageFile, paths.rootFSArchive, paths.metadata)
		if err != nil {
			return err
		}
	}

	if !cache.hasCacheDir {
		err = os.MkdirAll(imageCachePath, modeDirStd)
		if err != nil {
			return fmt.Errorf("unable to create image cache directory: %w", err)
		}
	}

	gfs, err = guestfs.Create()
	if err != nil {
		return err
	}
	defer func() {
		guestfsErr := gfs.Close()
		if guestfsErr != nil && err == nil {
			err = guestfsErr
		}
	}()

	err = b.setupVMImageFile(*gfs, paths.vmImageFile)
	if err != nil {
		return err
	}

	ctrImage, err := remote.Image(ctrImageRef)
	if err != nil {
		return fmt.Errorf("unable to retrieve remote image: %w", err)
	}

	err = b.setupRootFSArchive(ctrImage, gfs, paths.rootFSArchive)
	if err != nil {
		return err
	}

	err = b.setupBootloader(gfs)
	if err != nil {
		return err
	}

	err = b.setupInit(gfs)
	if err != nil {
		return err
	}

	err = b.setupKernel(gfs)
	if err != nil {
		return err
	}

	err = b.setupMetadata(ctrImage, gfs, paths.metadata)
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
options rw root=PARTUUID=%s console=ttyS0,115200 earlyprintk=ttyS0,115200 consoleblank=0 ip=dhcp init=%s/init
`
	return fmt.Sprintf(contentFmt, b.kernelVersion, partUUID, dirCB)
}

func (b *Builder) setupVMImageFile(gfs guestfs.Guestfs, vmImageFile string) error {
	err := gfs.Disk_create(vmImageFile, fmtRaw, tenGigs, &guestfs.OptargsDisk_create{
		Preallocation_is_set: true,
		Preallocation:        "sparse",
	})
	if err != nil {
		return fmt.Errorf("unable to create VM image file: %w", err)
	}

	err = gfs.Add_drive(vmImageFile, &guestfs.OptargsAdd_drive{
		Format_is_set: true, Format: fmtRaw,
	})
	if err != nil {
		return fmt.Errorf("unable to define disk drive in VM image: %w", err)
	}

	err = gfs.Launch()
	if err != nil {
		return fmt.Errorf("unable to launch backend to configure VM image: %w", err)
	}

	err = gfs.Part_init(deviceDisk, "gpt")

	// EFI Partition. Sectors 2048 through 501760 == 256MB.
	err = gfs.Part_add(deviceDisk, partTypePrimary, 2048, 501760)
	if err != nil {
		return fmt.Errorf("unable to create EFI partition in VM image: %w", err)
	}

	err = gfs.Part_set_bootable(deviceDisk, 1, true)
	if err != nil {
		return fmt.Errorf("unable to set EFI partition bootable in VM image: %w", err)
	}

	err = gfs.Part_set_name(deviceDisk, 1, "efi")
	if err != nil {
		return fmt.Errorf("unable to set name of EFI partition in VM image: %w", err)
	}

	err = gfs.Mkfs(FSTypeVFAT, devicePartEFI, &guestfs.OptargsMkfs{
		Label_is_set: true,
		Label:        labelEFI,
	})
	if err != nil {
		return fmt.Errorf("unable to format EFI partition in VM image: %w", err)
	}

	// Root partition. Sectors 501761 through 20971486 == remainder of disk.
	err = gfs.Part_add(deviceDisk, partTypePrimary, 501761, 20971486)
	if err != nil {
		return fmt.Errorf("unable to create root partition in VM image: %w", err)
	}

	err = gfs.Part_set_name(deviceDisk, 2, "root")
	if err != nil {
		return fmt.Errorf("unable to set name of root partition in VM image: %w", err)
	}

	err = gfs.Mkfs(FSTypeExt4, devicePartRoot, &guestfs.OptargsMkfs{
		Label_is_set: true,
		Label:        labelRoot,
	})
	if err != nil {
		return fmt.Errorf("unable to format root partition in VM image: %w", err)
	}

	err = gfs.Mount(devicePartRoot, dirRoot)
	if err != nil {
		return fmt.Errorf("unable to mount root partition in VM image: %w", err)
	}

	err = gfs.Mkdir_mode(dirEFI, modeDirStd)
	if err != nil {
		return fmt.Errorf("unable to create EFI boot directory in VM image: %w", err)
	}

	err = gfs.Mount(devicePartEFI, dirEFI)
	if err != nil {
		return fmt.Errorf("unable to mount EFI partition in VM image: %w", err)
	}

	err = gfs.Mkdir_mode(dirCB, modeDirStd)
	if err != nil {
		return fmt.Errorf("unable to create init directory in VM image: %w", err)
	}

	return nil
}

func (b *Builder) setupRootFSArchive(ctrImage v1.Image, gfs *guestfs.Guestfs, rootFSArchivePath string) (err error) {
	ctrImageReader := mutate.Extract(ctrImage)
	defer func() {
		rdrErr := ctrImageReader.Close()
		if rdrErr != nil && err == nil {
			err = rdrErr
		}
	}()

	// Write from imageReader to a local file. Unfortunately there is no way to write
	// directly into the VM image from the imageReader with the libguestfs API.
	rootFSArchive, err := os.Create(rootFSArchivePath)
	if err != nil {
		return fmt.Errorf("unable create container root filesystem archive: %w", err)
	}

	_, err = io.Copy(rootFSArchive, ctrImageReader)
	if err != nil {
		return fmt.Errorf("unable to write container root filesystem archive: %w", err)
	}

	err = rootFSArchive.Close()
	if err != nil {
		return fmt.Errorf("unable to close container root filesystem archive: %w", err)
	}

	err = gfs.Tar_in(rootFSArchivePath, dirRoot, nil)
	if err != nil {
		return fmt.Errorf("unable to copy container root filesystem into VM image: %w", err)
	}

	return nil
}

func (b *Builder) setupBootloader(gfs *guestfs.Guestfs) error {
	err := gfs.Tar_in(b.BootloaderPath, dirRoot, nil)
	if err != nil {
		return fmt.Errorf("unable to copy bootloader into VM image: %w", err)
	}

	partRootUUID, err := gfs.Part_get_gpt_guid(deviceDisk, 2)
	if err != nil {
		return fmt.Errorf("unable to get UUID of root partition in VM image: %w", err)
	}

	bootEntry := b.formatBootEntry(partRootUUID)

	err = gfs.Write_file("/boot/loader/entries/cb.conf", bootEntry, len(bootEntry))
	if err != nil {
		return fmt.Errorf("unable to write bootloader entry in VM image: %w", err)
	}

	err = gfs.Chmod(0644, "/boot/loader/entries/cb.conf")
	if err != nil {
		return fmt.Errorf("unable to set permissions of bootloader entry in VM image: %w", err)
	}

	return nil
}

func gfsCopyIn(gfs *guestfs.Guestfs, src, dest string, owner, group, mode int) error {
	destDir := filepath.Dir(dest)
	err := gfs.Copy_in(src, destDir)
	if err != nil {
		return fmt.Errorf("unable to copy local %s to %s in VM image: %w", src, dest, err)
	}

	srcBase := filepath.Base(src)
	destBase := filepath.Base(dest)
	if srcBase != destBase {
		// gfs.Copy_in takes only a destination directory, so it is necessary to move
		// the file if the source is not named the same as the destination file.
		err = gfs.Mv(filepath.Join(destDir, srcBase), dest)
		if err != nil {
			return fmt.Errorf("unable to move file to destination in VM image: %w", err)
		}
	}

	err = gfs.Chown(owner, group, dest)
	if err != nil {
		return fmt.Errorf("unable to change ownership of %s in VM image: %w", dest, err)
	}

	err = gfs.Chmod(mode, dest)
	if err != nil {
		return fmt.Errorf("unable to set permissions of %s in VM image: %w", dest, err)
	}

	return nil
}

func (b *Builder) setupInit(gfs *guestfs.Guestfs) error {
	return gfsCopyIn(gfs, b.InitPath, filepath.Join(dirCB, "init"), 0, 0, 0755)
}

func (b *Builder) setupKernel(gfs *guestfs.Guestfs) error {
	err := gfs.Tar_in(b.KernelPath, dirRoot, nil)
	if err != nil {
		return fmt.Errorf("unable to copy kernel into VM image: %w", err)
	}

	modExists, err := gfs.Exists(dirLibModules)
	if err != nil {
		return err
	}

	if !modExists {
		err = gfs.Mkdir_mode(dirLibModules, modeDirStd)
		if err != nil {
			return fmt.Errorf("unable to create directory for modules: %w", err)
		}
	}

	err = gfs.Ln_s(path.Join(dirCB, dirLibModules, b.kernelVersion), path.Join(dirLibModules, b.kernelVersion))
	if err != nil {
		return fmt.Errorf("unable to link modules in VM image: %w", err)
	}

	return nil
}

func (b *Builder) setupMetadata(ctrImage v1.Image, gfs *guestfs.Guestfs, metadataPath string) (err error) {
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

	err = gfsCopyIn(gfs, metadataPath, filepath.Join(dirCB, fileMetadata), 0, 0, 0644)
	if err != nil {
		return fmt.Errorf("unable to copy metadata into VM image: %w", err)
	}

	return nil
}

type ts struct {
	atime time.Time
	mtime time.Time
}

func (b *Builder) untar(reader io.Reader, verbose bool) error {
	oldUmask := syscall.Umask(0)
	defer func() {
		syscall.Umask(oldUmask)
	}()

	tss := map[string]ts{}

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
		dest := filepath.Join(dirRoot, hdr.Name)
		tss[dest] = ts{atime: hdr.AccessTime, mtime: hdr.ModTime}

		switch hdr.Typeflag {
		case tar.TypeBlock:
			if verbose {
				fmt.Printf("Creating block device %s, major: %d, minor: %d\n", dest,
					hdr.Devmajor, hdr.Devminor)
			}
			err = b.guestfs.Mknod_b(int(hdr.Mode), int(hdr.Devmajor), int(hdr.Devminor), dest)
			if err != nil {
				return newErrExtract(tar.TypeBlock, err)
			}
		case tar.TypeChar:
			if verbose {
				fmt.Printf("Creating character device %s, major: %d, minor: %d\n", dest,
					hdr.Devmajor, hdr.Devminor)
			}
			err = b.guestfs.Mknod_c(int(hdr.Mode), int(hdr.Devmajor), int(hdr.Devminor), dest)
			if err != nil {
				return newErrExtract(tar.TypeChar, err)
			}
		case tar.TypeDir:
			if verbose {
				fmt.Printf("Creating directory %s with mode %s\n", dest, os.FileMode(hdr.Mode))
			}
			err = b.guestfs.Mkdir_mode(dest, int(hdr.Mode))
			if err != nil {
				gErr, _ := err.(*guestfs.GuestfsError)
				if gErr.Errno == syscall.EEXIST {
					continue
				}
				return newErrExtract(tar.TypeDir, err)
			}
		case tar.TypeFifo:
			if verbose {
				fmt.Printf("Creating fifo %s with mode %s\n", dest, os.FileMode(hdr.Mode))
			}
			err = b.guestfs.Mkfifo(int(hdr.Mode), dest)
			if err != nil {
				return newErrExtract(tar.TypeFifo, err)
			}
		case tar.TypeLink:
			if verbose {
				fmt.Printf("Creating hard link %s with target %s\n", hdr.Linkname, dest)
			}
			// Note: the archive/tar package calls the link name
			// what libguestfs calls the target and vice versa.
			err = b.guestfs.Ln(hdr.Linkname, dest)
			if err != nil {
				gErr, _ := err.(*guestfs.GuestfsError)
				if gErr.Errno == syscall.EXDEV {
					// Got a cross-device link error...
					// Check if path is absolute, if not try to make it so.
					if !strings.HasPrefix(hdr.Linkname, dirRoot) {
						err = b.guestfs.Ln(filepath.Join(dirRoot, hdr.Linkname), dest)
						if err != nil {
							return newErrExtract(tar.TypeLink, err)
						}
					} else {
						// Fall back to creating a symlink.
						err = b.guestfs.Ln_s(filepath.Join(dirRoot, hdr.Linkname), dest)
						if err != nil {
							return fmt.Errorf("unable to create either hard or soft link: %w", err)
						}
					}
					continue
				}
				return newErrExtract(tar.TypeLink, err)
			}
		case tar.TypeReg:
			if verbose {
				fmt.Printf("Copying file %s with mode %s\n", dest, os.FileMode(hdr.Mode))
			}
			err = b.copyFile(treader, dest, fi.Mode())
			if err != nil {
				return newErrExtract(tar.TypeReg, err)
			}
		case tar.TypeSymlink:
			if verbose {
				fmt.Printf("Creating symbolic link %s with target %s\n", hdr.Linkname, dest)
			}
			err = b.guestfs.Ln_s(hdr.Linkname, dest)
			if err != nil {
				return newErrExtract(tar.TypeSymlink, err)
			}
		}

		err = b.guestfs.Lchown(hdr.Uid, hdr.Gid, dest)
		if err != nil {
			return err
		}
	}

	// Change all the timestamps at the end, because creation
	// of entries within directories resets parent timestamps.
	for dest, ts := range tss {
		// err := b.guestfs.Utimens(dest, ts.atime.Unix(),
		// 	ts.atime.UnixNano(), ts.mtime.Unix(), ts.mtime.UnixNano())
		err := b.guestfs.Utimens(dest, ts.atime.Unix(),
			-2, ts.mtime.Unix(), -2)
		if err != nil {
			return err
		}
	}

	return nil
}

// copyFile is slow and hacky because it relies on writing a file to local disk and
// then using libguestfs' cp_a() method to copy it into the image. Unfortunately this
// is necessary because the API does not provide a way to copy from a Reader.
func (b *Builder) copyFile(src io.Reader, dest string, perm os.FileMode) (err error) {
	var tempFile *os.File

	tempFile, err = os.CreateTemp(b.WorkingDir, "file")
	if err != nil {
		return fmt.Errorf("unable to create temp file for %s: %w", dest, err)
	}

	tempName := tempFile.Name()

	_, err = io.Copy(tempFile, src)
	if err != nil {
		return fmt.Errorf("unable to copy temp file %s for %s: %w", tempName, dest, err)
	}
	err = tempFile.Close()
	if err != nil {
		return fmt.Errorf("unable to close temp file %s for %s: %w", tempName, dest, err)
	}

	err = b.guestfs.Upload(tempName, dest)
	if err != nil {
		return fmt.Errorf("unable to copy temp file %s into image at %s: %w", tempName, dest, err)
	}

	err = b.guestfs.Chmod(int(perm), dest)
	if err != nil {
		return fmt.Errorf("unable to set permissions on temp file %s for %s: %w", tempName, dest, err)
	}

	return nil
}
