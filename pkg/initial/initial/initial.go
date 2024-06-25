package initial

import (
	"bufio"
	_ "crypto/sha256" // For JSON decoder.
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/cloudboss/easyto/pkg/initial/aws"
	"github.com/cloudboss/easyto/pkg/initial/service"
	"github.com/cloudboss/easyto/pkg/initial/vmspec"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

const (
	fileCACerts    = "amazon.pem"
	filePasswd     = "/etc/passwd"
	fileGroup      = "/etc/group"
	fileMetadata   = "metadata.json"
	fileMounts     = "/proc/mounts"
	dirRoot        = "/"
	execBits       = 0111
	pathEnvDefault = "/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin"
)

type link struct {
	target string
	path   string
}

type mount struct {
	source  string
	flags   uintptr
	fsType  string
	mode    os.FileMode
	options []string
	target  string
}

func mounts() error {
	ms := []mount{
		{
			source: "devpts",
			flags:  syscall.MS_NOATIME | syscall.MS_NOEXEC | syscall.MS_NOSUID,
			fsType: "devpts",
			mode:   0755,
			options: []string{
				"mode=0620",
				"gid=5",
				"ptmxmode=666",
			},
			target: "/dev/pts",
		},
		{
			source: "mqueue",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID,
			fsType: "mqueue",
			mode:   0755,
			target: "/dev/mqueue",
		},
		{
			source: "tmpfs",
			flags:  syscall.MS_NODEV | syscall.MS_NOSUID,
			fsType: "tmpfs",
			mode:   0777 | fs.ModeSticky,
			target: "/dev/shm",
		},
		{
			source: "hugetlbfs",
			flags:  syscall.MS_RELATIME,
			fsType: "hugetlbfs",
			mode:   0755,
			target: "/dev/hugepages",
		},
		{
			source: "proc",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "proc",
			mode:   0555,
			target: "/proc",
		},
		{
			source: "sys",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID,
			fsType: "sysfs",
			mode:   0555,
			target: "/sys",
		},
		{
			source: "tmpfs",
			flags:  syscall.MS_NODEV | syscall.MS_NOSUID,
			fsType: "tmpfs",
			mode:   0755,
			options: []string{
				"mode=0755",
			},
			target: constants.DirRun,
		},
		{
			mode:   0777 | fs.ModeSticky,
			target: filepath.Join(constants.DirRun, "lock"),
		},
		{
			source: "tmpfs",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID,
			fsType: "tmpfs",
			options: []string{
				"mode=0755",
			},
			target: "/sys/fs/cgroup",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"net_cls",
				"net_prio",
			},
			target: "/sys/fs/cgroup/net_cls,net_prio",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"hugetlb",
			},
			target: "/sys/fs/cgroup/hugetlb",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"pids",
			},
			target: "/sys/fs/cgroup/pids",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"freezer",
			},
			target: "/sys/fs/cgroup/freezer",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"cpu",
				"cpuacct",
			},
			target: "/sys/fs/cgroup/cpu,cpuacct",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"devices",
			},
			target: "/sys/fs/cgroup/devices",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"blkio",
			},
			target: "/sys/fs/cgroup/blkio",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"memory",
			},
			target: "/sys/fs/cgroup/memory",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"perf_event",
			},
			target: "/sys/fs/cgroup/perf_event",
		},
		{
			source: "cgroup",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "cgroup",
			mode:   0555,
			options: []string{
				"cpuset",
			},
			target: "/sys/fs/cgroup/cpuset",
		},
		{
			source: "nodev",
			fsType: "debugfs",
			mode:   0500,
			target: "/sys/kernel/debug",
		},
	}

	// Temporarily unset umask to ensure directory modes are exactly as configured.
	oldUmask := syscall.Umask(0)
	defer func() {
		syscall.Umask(oldUmask)
	}()

	for _, m := range ms {
		slog.Debug("About to process mount", "mount", m)
		_, err := os.Stat(m.target)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("unexpected error checking status of %s: %w", m.target, err)
			}
			slog.Debug("About to make directory", "directory", m.target, "mode", m.mode)
			err := os.MkdirAll(m.target, m.mode)
			if err != nil {
				return fmt.Errorf("unable to create directory %s: %w", m.target, err)
			}
		}
		skipMount := len(m.fsType) == 0
		if skipMount {
			continue
		}
		err = unix.Mount(m.source, m.target, m.fsType, m.flags, strings.Join(m.options, ","))
		if err != nil {
			return fmt.Errorf("unable to mount %s on %s: %w", m.source, m.target, err)
		}
	}
	return nil
}

func links() error {
	ls := []link{
		{
			target: "/proc/self/fd",
			path:   "/dev/fd",
		},
		{
			target: "/proc/self/fd/0",
			path:   "/dev/stdin",
		},
		{
			target: "/proc/self/fd/1",
			path:   "/dev/stdout",
		},
		{
			target: "/proc/self/fd/2",
			path:   "/dev/stderr",
		},
	}
	for _, l := range ls {
		err := os.Symlink(l.target, l.path)
		if err != nil {
			return fmt.Errorf("unable to symlink %s to %s: %w", l.path, l.target, err)
		}
	}
	return nil
}

func debug() {
	commands := [][]string{
		{
			"/bin/lsmod",
		},
		{
			"/bin/mount",
		},
		{
			"/bin/ps",
			"-ef",
		},
		{
			"/bin/ls",
			"-l",
			dirRoot,
		},
		{
			"/bin/ls",
			"-l",
			filepath.Join(dirRoot, "dev"),
		},
	}
	for _, command := range commands {
		args := []string{}
		if len(command) > 0 {
			args = command[1:]
		}
		err := runCommand(command[0], args...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running '%s': %s\n",
				strings.Join(command, " "), err)
		}
	}

}

func runCommand(executable string, args ...string) error {
	cmd := exec.Command(executable, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getenv gets the value of an environment variable from the environment
// passed in env, rather than the process's environment as with os.Getenv.
func getenv(env []string, key string) string {
	for _, envVar := range env {
		fields := strings.Split(envVar, "=")
		if fields[0] == key {
			return strings.Join(fields[1:], "=")
		}
	}
	return ""
}

func findExecutableInPath(executable, pathEnv string) (string, error) {
	for _, dir := range filepath.SplitList(pathEnv) {
		findPath := filepath.Join(dirRoot, dir, executable)
		fi, err := os.Stat(findPath)
		if err != nil {
			continue
		}
		if fi.Mode()&execBits != 0 {
			return filepath.Join(dir, executable), nil
		}
	}
	return "", fmt.Errorf("executable %s not found", executable)
}

func readMetadata(metadataPath string) (*v1.ConfigFile, error) {
	f, err := os.Open(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", metadataPath, err)
	}
	defer f.Close()

	metadata := &v1.ConfigFile{}
	err = json.NewDecoder(f).Decode(metadata)
	if err != nil {
		return nil, fmt.Errorf("unable to decode metadata: %w", err)
	}

	return metadata, nil
}

func fullCommand(spec *vmspec.VMSpec) ([]string, error) {
	ex := append(spec.Command, spec.Args...)
	if ex == nil {
		ex = []string{"/bin/sh"}
	}

	pathEnv := pathEnvDefault
	if pathVMSpec, i := spec.Env.Find("PATH"); i >= 0 {
		pathEnv = pathVMSpec
	}

	if !strings.HasPrefix(ex[0], dirRoot) {
		executablePath, err := findExecutableInPath(ex[0], pathEnv)
		if err != nil {
			return nil, err
		}
		ex[0] = executablePath
	}
	return ex, nil
}

// envToEnv converts an array of "key=value" strings to a NameValueSource.
func envToEnv(envVars []string) (vmspec.NameValueSource, error) {
	source := make(vmspec.NameValueSource, len(envVars))
	for i, envVar := range envVars {
		fields := strings.Split(envVar, "=")
		if len(fields) < 1 {
			return nil, fmt.Errorf("invalid environment variable '%s'", envVar)
		}
		source[i].Name = fields[0]
		if len(fields) > 1 {
			source[i].Value = strings.Join(fields[1:], "=")
		}
	}
	return source, nil
}

func metadataToVMSpec(metadata *v1.ConfigFile) (*vmspec.VMSpec, error) {
	spec := &vmspec.VMSpec{
		Command:             metadata.Config.Entrypoint,
		Args:                metadata.Config.Cmd,
		ShutdownGracePeriod: 10,
		Security:            vmspec.SecurityContext{},
	}

	env, err := envToEnv(metadata.Config.Env)
	if err != nil {
		return nil, err
	}
	spec.Env = env

	spec.WorkingDir = metadata.Config.WorkingDir
	if len(spec.WorkingDir) == 0 {
		spec.WorkingDir = dirRoot
	}

	uid, gid, err := getUserGroup(metadata.Config.User)
	if err != nil {
		return nil, err
	}
	spec.Security.RunAsUserID = uid
	spec.Security.RunAsGroupID = gid

	return spec, nil
}

// entryID parses entryFile in the format of /etc/passwd or /etc/group to get
// the numeric ID for the given entry. The entryFile has fields delimited by `:`
// characters; the first field is the entry (user or group name as a string), and
// the third field is its numeric ID. Additional fields are ignored, so it is able
// to parse /etc/passwd and /etc/group, although their number of fields differ.
// The function returns the numeric ID or an error if it is not found.
func entryID(entryFile, entry string) (int, error) {
	id := 0

	p, err := os.Open(entryFile)
	if err != nil {
		return 0, fmt.Errorf("unable to open %s: %w", entryFile, err)
	}
	defer p.Close()

	entryFound := false
	pScanner := bufio.NewScanner(p)
	for pScanner.Scan() {
		line := pScanner.Text()
		fields := strings.Split(line, ":")
		if fields[0] == entry {
			entryFound = true
			id, err = strconv.Atoi(fields[2])
			if err != nil {
				return 0, fmt.Errorf("unexpected error reading %s: %w",
					entryFile, err)
			}
			break
		}
	}
	if err = pScanner.Err(); err != nil {
		return 0, fmt.Errorf("unable to read %s: %w", entryFile, err)
	}
	if !entryFound {
		return 0, fmt.Errorf("%s not found in %s", entry, entryFile)
	}

	return id, nil
}

func getUserGroup(userEntry string) (int, int, error) {
	var (
		user  string
		group string
		uid   int
		gid   int
		err   error
	)

	userEntryFields := strings.Split(userEntry, ":")
	if len(userEntryFields) == 1 {
		user = userEntryFields[0]
	}

	if len(user) == 0 || user == "root" {
		return 0, 0, nil
	}

	if len(userEntryFields) == 2 {
		group = userEntryFields[1]
	}

	uid, err = entryID(filePasswd, user)
	if err != nil {
		return 0, 0, err
	}

	if len(group) == 0 || group == "root" {
		return uid, gid, nil
	}

	gid, err = entryID(fileGroup, group)
	if err != nil {
		return 0, 0, err
	}

	return uid, gid, nil
}

func parseMode(mode string) (fs.FileMode, error) {
	if len(mode) == 0 {
		return 0755, nil
	}
	n, err := strconv.ParseInt(mode, 8, 0)
	if err != nil {
		return 0, fmt.Errorf("invalid mode %s", mode)
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid mode %s", mode)
	}
	return fs.FileMode(n), nil
}

func handleVolumeEBS(volume *vmspec.EBSVolumeSource, index int) error {
	slog.Debug("Handling volume", "volume", volume)

	if len(volume.Device) == 0 {
		return errors.New("volume must have device")
	}

	if len(volume.FSType) == 0 {
		return errors.New("volume must have filesystem type")
	}

	if len(volume.Mount.Directory) == 0 {
		return errors.New("volume must have mount point")
	}

	mode, err := parseMode(volume.Mount.Mode)
	if err != nil {
		return err
	}
	slog.Debug("Parsed mode", "before", volume.Mount.Mode, "mode", mode)

	err = os.MkdirAll(volume.Mount.Directory, mode)
	if err != nil {
		return fmt.Errorf("unable to create mount point %s: %w",
			volume.Mount.Directory, err)
	}
	slog.Debug("Created mount point", "directory", volume.Mount.Directory)

	err = os.Chown(volume.Mount.Directory, volume.Mount.UserID,
		volume.Mount.GroupID)
	if err != nil {
		return fmt.Errorf("unable to change ownership of mount point: %w", err)
	}
	slog.Debug("Changed ownership of mount point", "directory", volume.Mount.Directory)

	hasFS, err := deviceHasFS(filepath.Join(constants.DirETSbin, "blkid"), volume.Device)
	if err != nil {
		return fmt.Errorf("unable to determine if %s has a filesystem: %w", volume.Device, err)
	}
	if !hasFS {
		mkfsPath := filepath.Join(constants.DirETSbin, "mkfs."+volume.FSType)
		if _, err := os.Stat(mkfsPath); os.IsNotExist(err) {
			return fmt.Errorf("unsupported filesystem type %s for volume at index %d",
				volume.FSType, index)
		}
		err = runCommand(mkfsPath, volume.Device)
		if err != nil {
			return fmt.Errorf("unable to create filesystem on %s: %w", volume.Device, err)
		}
		slog.Debug("Created filesystem", "device", volume.Device, "fstype", volume.FSType)
	}

	err = unix.Mount(volume.Device, volume.Mount.Directory, volume.FSType, 0, "")
	if err != nil {
		return fmt.Errorf("unable to mount %s on %s: %w", volume.Mount.Directory,
			volume.FSType, err)
	}
	slog.Debug("Mounted volume", "device", volume.Device, "directory", volume.Mount.Directory)

	return nil
}

func handleVolumeSSMParameter(volume *vmspec.SSMParameterVolumeSource, uid, gid int, conn aws.Connection) error {
	parameters, err := conn.SSMClient().GetParameters(volume.Path)
	if !(err == nil || volume.Optional) {
		return err
	}
	if volume.Mount.UserID != 0 {
		uid = volume.Mount.UserID
	}
	if volume.Mount.GroupID != 0 {
		gid = volume.Mount.GroupID
	}
	if err == nil {
		return parameters.Write(volume.Mount.Directory, "", uid, gid)
	}
	return nil
}

func handleVolumeS3(volume *vmspec.S3VolumeSource, conn aws.Connection) error {
	s3Client := conn.S3Client()
	objects, err := s3Client.ListObjects(volume.Bucket, volume.KeyPrefix)
	if !(err == nil || volume.Optional) {
		return err
	}
	if err == nil {
		return s3Client.CopyObjects(objects, volume.Mount.Directory, "", volume.Mount.UserID,
			volume.Mount.GroupID)
	}
	return nil
}

func doExec(spec *vmspec.VMSpec, command []string, env vmspec.NameValueSource) error {
	err := os.Chdir(spec.WorkingDir)
	if err != nil {
		return fmt.Errorf("unable to change working directory to %s: %w",
			spec.WorkingDir, err)
	}

	err = syscall.Setgid(spec.Security.RunAsGroupID)
	if err != nil {
		return fmt.Errorf("unable to set GID: %w", err)
	}

	err = syscall.Setuid(spec.Security.RunAsUserID)
	if err != nil {
		return fmt.Errorf("unable to set UID: %w", err)
	}

	return syscall.Exec(command[0], command, env.ToStrings())
}

func doForkExec(spec *vmspec.VMSpec, command []string, env vmspec.NameValueSource) error {
	supervisor := &service.Supervisor{
		Main: service.NewMainService(
			command,
			env.ToStrings(),
			spec.WorkingDir,
			uint32(spec.Security.RunAsGroupID),
			uint32(spec.Security.RunAsUserID),
		),
		Timeout: time.Duration(spec.ShutdownGracePeriod) * time.Second,
	}
	err := supervisor.Start()
	if err != nil {
		return fmt.Errorf("unable to start supervisor: %w", err)
	}

	waitForShutdown(spec, supervisor)
	return nil
}

func waitForShutdown(spec *vmspec.VMSpec, supervisor *service.Supervisor) {
	supervisor.Wait()

	mountPoints := spec.Volumes.MountPoints()
	osFS := afero.NewOsFs()

	err := unmountAll(osFS, mountPoints)
	if err != nil {
		slog.Error("Error unmounting volumes", "error", err)
	}

	// Best-effort wait, even if there were unmount errors. This can be improved
	// so it doesn't wait unnecessarily if no calls to unmount succeeded.
	waitForUnmounts(osFS, fileMounts, mountPoints, 10*time.Second)
}

// unmountAll remounts / as readonly and lazily unmounts all the volumes in the list of mount points.
func unmountAll(fs afero.Fs, mountPoints []string) error {
	var errs error

	err := unix.Mount("", dirRoot, "", syscall.MS_REMOUNT|syscall.MS_RDONLY, "")
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("unable to remount / as read-only: %w", err))
	}

	for _, mountPoint := range mountPoints {
		err := syscall.Unmount(mountPoint, syscall.MNT_DETACH)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable to unmount %s: %w", mountPoint, err))
		}
	}

	syscall.Sync()

	return errs
}

func waitForUnmounts(fs afero.Fs, mtab string, mountPoints []string, timeout time.Duration) {
	unmounted := map[string]struct{}{}
	end := time.Now().Add(timeout)

loop:
	slog.Debug("Waiting for unmounts", "mountpoints", mountPoints)
	for _, mountPoint := range mountPoints {
		mounted, err := isMounted(fs, mountPoint, mtab)
		if err != nil {
			slog.Error("Unable to check if mount point is mounted", "mountpoint", mountPoint, "error", err)
		}
		if !mounted {
			unmounted[mountPoint] = struct{}{}
			slog.Debug("Mount point is unmounted", "mountpoint", mountPoint)
		}
	}

	now := time.Now()
	lenUnmounted := len(unmounted)
	lenMountPoints := len(mountPoints)

	if now.Before(end) && lenUnmounted < lenMountPoints {
		goto loop
	}

	if now.After(end) && lenUnmounted < lenMountPoints {
		slog.Error("Timeout waiting for unmounts")
		return
	}

	slog.Info("All mount points unmounted")
}

func isMounted(fs afero.Fs, mountPoint, mtab string) (bool, error) {
	f, err := fs.Open(mtab)
	if err != nil {
		return false, fmt.Errorf("unable to open %s: %w", mtab, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return false, fmt.Errorf("invalid line in %s: %s", mtab, line)
		}
		mtabMountPoint := fields[1]
		if mtabMountPoint == mountPoint {
			return true, nil
		}
	}

	return false, err
}

func resolveAllEnvs(conn aws.Connection, env vmspec.NameValueSource,
	envFrom vmspec.EnvFromSource) (vmspec.NameValueSource, error) {
	var (
		errs        error
		resolvedEnv vmspec.NameValueSource
	)

	for _, e := range envFrom {
		if e.SSMParameter != nil {
			parameters, err := conn.SSMClient().GetParameters(e.SSMParameter.Path)
			if !(err == nil || e.SSMParameter.Optional) {
				errs = errors.Join(errs, err)
			}
			if err == nil {
				// Use ToMapString() to filter out any nested
				// paths below e.SSMParameter.Path.
				for k, v := range parameters.ToMapString() {
					ev := vmspec.NameValue{Name: k, Value: v}
					resolvedEnv = append(resolvedEnv, ev)
				}
			}
		}
	}
	if errs != nil {
		return nil, errs
	}

	lenEnv := len(env)
	allEnv := make(vmspec.NameValueSource, lenEnv+len(resolvedEnv))

	for i, e := range env {
		allEnv[i] = e
	}

	for i, e := range resolvedEnv {
		allEnv[lenEnv+i] = e
	}

	return allEnv, nil
}

func Run() error {
	slog.Info("Starting init")

	// Override Go's builtin known certificate directories, for
	// making API calls to AWS.
	os.Setenv("SSL_CERT_FILE", filepath.Join(constants.DirETEtc, fileCACerts))

	err := mounts()
	if err != nil {
		return err
	}

	err = links()
	if err != nil {
		return err
	}

	linkEBSDevicesErrC := make(chan error, 1)
	go func() {
		linkEBSDevicesErrC <- linkEBSDevices()
	}()

	metadata, err := readMetadata(filepath.Join(constants.DirETRoot, fileMetadata))
	if err != nil {
		return err
	}

	spec, err := metadataToVMSpec(metadata)
	if err != nil {
		return err
	}

	userData, err := aws.GetUserData()
	if err != nil {
		return fmt.Errorf("unable to get user data: %w", err)
	}

	err = spec.Merge(userData)
	if err != nil {
		return fmt.Errorf("unable to merge VMSpec with user data: %w", err)
	}

	if spec.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	err = spec.Validate()
	if err != nil {
		return fmt.Errorf("user data failed to validate: %w", err)
	}

	err = SetSysctls(spec.Sysctls)
	if err != nil {
		return err
	}

	region, err := aws.GetRegion()
	if err != nil {
		return err
	}

	conn, err := aws.NewConnection(region)
	if err != nil {
		return err
	}

	// Ensure linkEBSDevices() is done before handling volumes.
	err = <-linkEBSDevicesErrC
	if err != nil {
		return err
	}

	err = resizeRootVolume()
	if err != nil {
		return err
	}

	for i, volume := range spec.Volumes {
		if volume.EBS != nil {
			err = handleVolumeEBS(volume.EBS, i)
			if err != nil {
				return err
			}
		}
		if volume.SSMParameter != nil {
			err = handleVolumeSSMParameter(volume.SSMParameter, spec.Security.RunAsUserID,
				spec.Security.RunAsGroupID, conn)
			if err != nil {
				return err
			}
		}
		if volume.S3 != nil {
			err = handleVolumeS3(volume.S3, conn)
			if err != nil {
				return err
			}
		}
	}

	if spec.Security.ReadonlyRootFS {
		err = unix.Mount("", dirRoot, "", syscall.MS_REMOUNT|syscall.MS_RDONLY, "")
		if err != nil {
			return fmt.Errorf("unable to remount root as readonly: %w", err)
		}
	}

	command, err := fullCommand(spec)
	if err != nil {
		return err
	}

	env, err := resolveAllEnvs(conn, spec.Env, spec.EnvFrom)
	if err != nil {
		return fmt.Errorf("unable to resolve all environment variables: %w", err)
	}

	slog.Debug("About to run entrypoint", "command", command, "env", env)

	if spec.ReplaceInit {
		err = doExec(spec, command, env)
	} else {
		err = doForkExec(spec, command, env)
	}

	return err
}
