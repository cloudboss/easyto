package preinit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mdlayher/vsock"
	"golang.org/x/sys/unix"
)

const (
	agentPath       = "/cb_agent"
	configPath      = "/config"
	execBits        = 0111
	fsTypeExt4      = "ext4"
	koPath          = "/cbagent.ko"
	module          = "cbagent"
	modulesDevice   = "/dev/vdb"
	mountExecutable = "/bin/mount"
	pathDefault     = "/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin"
	rootDevice      = "/dev/vda"
	rootDir         = "/"
	rootMountTmp    = "/newroot"
)

var executableNotFound = errors.New("executable not found")

// This is a duplicate of the struct in github.com/cloudboss/cb/punk, as
// CGO_ENABLED=0 cannot be used with that library as it requires C.
type Fuse struct {
	Cmd        []string `json:"cmd,omitempty"`
	Entrypoint []string `json:"entrypoint,omitempty"`
	Env        []string `json:"env,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
}

type InitSpec struct {
	AbsPath    string
	Args       []string
	Env        []string
	GID        int
	UID        int
	WorkingDir string
}

type link struct {
	target string
	path   string
}

type mount struct {
	source  string
	flags   uintptr
	fsType  string
	mode    int
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
			source: "shm",
			flags:  syscall.MS_NODEV | syscall.MS_NOSUID,
			fsType: "tmpfs",
			mode:   1777,
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
			source: "binfmt_misc",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_RELATIME,
			fsType: "binfmt_misc",
			target: "/proc/sys/fs/binfmt_misc",
		},
		{
			source: "sys",
			flags:  syscall.MS_NODEV | syscall.MS_NOEXEC | syscall.MS_NOSUID,
			fsType: "sysfs",
			mode:   0555,
			target: "/sys",
		},
		{
			source: "run",
			flags:  syscall.MS_NODEV | syscall.MS_NOSUID,
			fsType: "tmpfs",
			mode:   0755,
			options: []string{
				"mode=0755",
			},
			target: "/run",
		},
		{
			mode:   1777,
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
	for _, m := range ms {
		fmt.Printf("About to process mount: %+v\n", m)
		_, err := os.Stat(m.target)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("unexpected error checking status of %s: %w", m.target, err)
			}
			fmt.Println("About to make directories")
			err := os.MkdirAll(m.target, fs.FileMode(m.mode))
			if err != nil {
				return fmt.Errorf("unable to create directory %s: %w", m.target, err)
			}
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

func runCommand(executable string, args ...string) error {
	cmd := exec.Command(executable, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func osRelease() (string, error) {
	utsname := unix.Utsname{}
	err := unix.Uname(&utsname)
	if err != nil {
		return "", err
	}
	i := bytes.IndexByte(utsname.Release[:], 0)
	return string(utsname.Release[:i]), nil
}

func modulesPath() (string, error) {
	rel, err := osRelease()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/lib/modules/%s", rel), nil
}

func modprobe(module string, args []string) error {
	m, err := os.Open(module)
	if err != nil {
		return err
	}
	return unix.FinitModule(int(m.Fd()), strings.Join(args, " "), 0)
}

func readConfig() (*Fuse, error) {
	cx, err := vsock.Dial(unix.VMADDR_CID_HOST, 9999, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to vsock: %w", err)
	}
	defer cx.Close()

	fmt.Println("attempting to read from socket")

	fuse := &Fuse{}
	err = json.NewDecoder(cx).Decode(fuse)
	if err != nil {
		return nil, err
	}
	return fuse, nil
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
			rootMountTmp,
		},
		{
			"/bin/ls",
			path.Join(rootMountTmp, "dev"),
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
		findPath := path.Join(rootMountTmp, dir, executable)
		fi, err := os.Stat(findPath)
		if err != nil {
			continue
		}
		if fi.Mode()&execBits != 0 {
			return path.Join(dir, executable), nil
		}
	}
	return "", executableNotFound
}

func Setup() (*InitSpec, error) {
	fmt.Println("This is the start of the setup")
	err := mounts()
	if err != nil {
		return nil, err
	}
	fmt.Println("After mounts()")
	err = links()
	if err != nil {
		return nil, err
	}
	fmt.Println("After links()")
	debug()
	fmt.Println("After debug()")
	fuse, err := readConfig()
	if err != nil {
		return nil, err
	}
	return FuseToInitSpec(fuse)
}

func FuseToInitSpec(fuse *Fuse) (*InitSpec, error) {
	fmt.Printf("fuse: %+v\n", fuse)
	if len(fuse.Entrypoint) == 0 && len(fuse.Cmd) == 0 {
		return nil, fmt.Errorf("a command is required")
	}

	args := fuse.Entrypoint
	args = append(args, fuse.Cmd...)

	pathEnv := pathDefault
	pathFuse := getenv(fuse.Env, "PATH")
	if len(pathFuse) > 0 {
		pathEnv = pathFuse
	}

	executable := args[0]
	if !strings.HasPrefix(executable, rootDir) {
		executablePath, err := findExecutableInPath(executable, pathEnv)
		if err != nil {
			return nil, err
		}
		executable = executablePath
		args[0] = executablePath
	}

	env := []string{fmt.Sprintf("PATH=%s", pathDefault)}
	if len(fuse.Env) > 0 {
		env = fuse.Env
	}

	workingDir := rootDir
	if len(fuse.WorkingDir) > 0 {
		workingDir = fuse.WorkingDir
	}

	initSpec := &InitSpec{
		AbsPath:    executable,
		Args:       args,
		Env:        env,
		WorkingDir: workingDir,
	}
	return initSpec, nil
}

func SwitchRoot(newRoot string, initSpec *InitSpec) error {
	fmt.Printf("initSpec: %+v\n", initSpec)

	err := os.Chdir(newRoot)
	if err != nil {
		return fmt.Errorf("unable to change to %s dir: %w",
			newRoot, err)
	}
	err = unix.Chroot(".")
	if err != nil {
		return fmt.Errorf("unable to chroot to %s dir: %w",
			newRoot, err)
	}
	err = os.Chdir(initSpec.WorkingDir)
	if err != nil {
		return fmt.Errorf("unable to change to %s dir in chroot: %w",
			initSpec.WorkingDir, err)
	}
	return syscall.Exec(initSpec.AbsPath, initSpec.Args, initSpec.Env)
}
