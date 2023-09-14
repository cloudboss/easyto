package preinit

import (
	"bufio"
	_ "crypto/sha256" // For JSON decoder.
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/cloudboss/easyto/preinit/aws"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"golang.org/x/sys/unix"
)

const (
	fileCACerts    = "amazon.pem"
	filePasswd     = "/etc/passwd"
	fileGroup      = "/etc/group"
	fileMetadata   = "metadata.json"
	dirRoot        = "/"
	dirCB          = "/__cb__"
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
			source: "devtmpfs",
			flags:  syscall.MS_NOSUID,
			fsType: "devtmpfs",
			mode:   0755,
			target: "/dev",
		},
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
			target: "/run",
		},
		{
			mode:   0777 | fs.ModeSticky,
			target: "/run/lock",
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
		fmt.Printf("About to process mount: %+v\n", m)
		_, err := os.Stat(m.target)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("unexpected error checking status of %s: %w", m.target, err)
			}
			fmt.Printf("About to make directory %s with mode %s\n", m.target, m.mode)
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
			"/dev",
		},
		{
			"/bin/ls",
			dirRoot,
		},
		{
			"/bin/ls",
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

func fullCommand(vmspec *VMSpec) ([]string, error) {
	ex := append(vmspec.Command, vmspec.Args...)
	if ex == nil {
		ex = []string{"/bin/sh"}
	}

	pathEnv := pathEnvDefault
	if pathVMSpec, i := vmspec.Env.find("PATH"); i >= 0 {
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

// envToEnv converts an array of "key=value" strings to an EnvVarSource.
func envToEnv(envVars []string) (EnvVarSource, error) {
	source := make(EnvVarSource, len(envVars))
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

func metadataToVMSpec(metadata *v1.ConfigFile) (*VMSpec, error) {
	vmSpec := &VMSpec{
		Command:  metadata.Config.Entrypoint,
		Args:     metadata.Config.Cmd,
		Security: SecurityContext{},
	}

	env, err := envToEnv(metadata.Config.Env)
	if err != nil {
		return nil, err
	}
	vmSpec.Env = env

	vmSpec.WorkingDir = metadata.Config.WorkingDir
	if len(vmSpec.WorkingDir) == 0 {
		vmSpec.WorkingDir = dirRoot
	}

	uid, gid, err := getUserGroup(metadata.Config.User)
	if err != nil {
		return nil, err
	}
	vmSpec.Security.RunAsUserID = uid
	vmSpec.Security.RunAsGroupID = gid

	return vmSpec, nil
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

func handleVolumeEBS(volume *EBSVolumeSource, index int) error {
	fmt.Printf("Handling volume: %+v\n", volume)

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
	fmt.Printf("Parsed mode %s into %s\n", volume.Mount.Mode, mode)

	err = os.MkdirAll(volume.Mount.Directory, mode)
	if err != nil {
		return fmt.Errorf("unable to create mount point %s: %w",
			volume.Mount.Directory, err)
	}
	fmt.Printf("Created mount point %s\n", volume.Mount.Directory)

	err = os.Chown(volume.Mount.Directory, volume.Mount.UserID,
		volume.Mount.GroupID)
	if err != nil {
		return fmt.Errorf("unable to change ownership of mount point: %w", err)
	}
	fmt.Printf("Changed ownership of mount point %s\n", volume.Mount.Directory)

	mkfsPath := filepath.Join(dirCB, "mkfs."+volume.FSType)
	if _, err := os.Stat(mkfsPath); os.IsNotExist(err) {
		return fmt.Errorf("unsupported filesystem type %s for volume at index %d",
			volume.FSType, index)
	}

	err = runCommand(mkfsPath, volume.Device)
	if err != nil {
		return fmt.Errorf("unable to create filesystem on %s: %w", volume.Device, err)
	}
	fmt.Printf("Created %s filesystem on %s\n", volume.FSType, volume.Device)

	err = unix.Mount(volume.Device, volume.Mount.Directory, volume.FSType, 0, "")
	if err != nil {
		return fmt.Errorf("unable to mount %s on %s: %w", volume.Mount.Directory,
			volume.FSType, err)
	}
	fmt.Printf("Mounted %s on %s\n", volume.Device, volume.Mount.Directory)

	return nil
}

func execCommand(vmspec *VMSpec) error {
	command, err := fullCommand(vmspec)
	if err != nil {
		return err
	}

	err = os.Chdir(vmspec.WorkingDir)
	if err != nil {
		return fmt.Errorf("unable to change working directory to %s: %w",
			vmspec.WorkingDir, err)
	}

	err = syscall.Setuid(vmspec.Security.RunAsUserID)
	if err != nil {
		return fmt.Errorf("unable to set UID: %w", err)
	}

	err = syscall.Setgid(vmspec.Security.RunAsGroupID)
	if err != nil {
		return fmt.Errorf("unable to set GID: %w", err)
	}

	region, err := getRegion()
	if err != nil {
		return err
	}

	conn, err := aws.NewConnection(region)
	if err != nil {
		return err
	}

	env, err := resolveAllEnvs(conn, vmspec.Env, vmspec.EnvFrom)
	if err != nil {
		return fmt.Errorf("unable to resolve all environment variables: %w", err)
	}

	return syscall.Exec(command[0], command, env.toStrings())
}

func resolveAllEnvs(conn aws.Connection, env EnvVarSource, envFrom EnvFromSource) (EnvVarSource, error) {
	var (
		errs        error
		resolvedEnv EnvVarSource
	)

	for _, e := range envFrom {
		if e.SSMParameter != nil {
			parameters, err := conn.SSMClient().GetParameters(e.SSMParameter.Path)
			if !(err == nil || e.SSMParameter.Optional) {
				errs = errors.Join(errs, err)
			}
			if err == nil {
				// Use mapAnyToMapString() to filter out any nested
				// paths below e.SSMParameter.Path.
				for k, v := range mapAnyToMapString(parameters) {
					ev := EnvVar{Name: k, Value: v}
					resolvedEnv = append(resolvedEnv, ev)
				}
			}
		}
	}
	if errs != nil {
		return nil, errs
	}

	lenEnv := len(env)
	allEnv := make(EnvVarSource, lenEnv+len(resolvedEnv))

	for i, e := range env {
		allEnv[i] = e
	}

	for i, e := range resolvedEnv {
		allEnv[lenEnv+i] = e
	}

	return allEnv, nil
}

func mapAnyToMapString(anyMap map[string]any) map[string]string {
	stringMap := map[string]string{}
	for k, v := range anyMap {
		switch v.(type) {
		case string:
			stringMap[k] = v.(string)
		}
	}
	return stringMap
}

func Run() error {
	fmt.Println("Starting init")

	// Override Go's builtin known certificate directories, for
	// making API calls to AWS.
	os.Setenv("SSL_CERT_FILE", filepath.Join(dirCB, fileCACerts))

	err := mounts()
	if err != nil {
		return err
	}
	fmt.Println("After mounts()")

	err = links()
	if err != nil {
		return err
	}
	fmt.Println("After links()")

	linkEBSDevicesErrC := make(chan error, 1)
	go linkEBSDevices(linkEBSDevicesErrC)

	debug()
	fmt.Println("After debug()")

	metadata, err := readMetadata(filepath.Join(dirCB, fileMetadata))
	if err != nil {
		return err
	}
	fmt.Println("After readMetadata()")

	vmSpec, err := metadataToVMSpec(metadata)
	if err != nil {
		return err
	}
	fmt.Println("After metadataToVMSpec()")

	userData, err := getUserData()
	if err != nil {
		return fmt.Errorf("unable to get user data: %w", err)
	}
	fmt.Println("After getUserData()")
	vmSpec = vmSpec.merge(userData)
	fmt.Println("After vmSpec.merge()")

	err = vmSpec.Validate()
	if err != nil {
		return fmt.Errorf("user data failed to validate: %w", err)
	}

	// Ensure linkEBSDevices() is done before handling volumes.
	err = <-linkEBSDevicesErrC
	if err != nil {
		return err
	}

	for i, volume := range vmSpec.Volumes {
		if volume.EBS != nil {
			err = handleVolumeEBS(volume.EBS, i)
			if err != nil {
				return err
			}
			continue
		}
		if volume.SecretsManager != nil {
			continue
		}
		return fmt.Errorf("invalid volume defined at index %d", i)
	}
	fmt.Println("After handling volumes")

	if vmSpec.Security.ReadonlyRootFS {
		err = unix.Mount("", dirRoot, "", syscall.MS_REMOUNT|syscall.MS_RDONLY, "")
		if err != nil {
			return fmt.Errorf("unable to remount root as readonly: %w", err)
		}
	}

	fmt.Println("About to run entrypoint")
	return execCommand(vmSpec)
}
