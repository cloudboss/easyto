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
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"golang.org/x/sys/unix"
)

const (
	filePasswd   = "/etc/passwd"
	fileGroup    = "/etc/group"
	fileMetadata = "metadata.json"
	dirRoot      = "/"
	dirCB        = "/__cb__"
	execBits     = 0111
	pathDefault  = "/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin"
)

// This is a duplicate of the struct in github.com/cloudboss/cb/punk, as
// CGO_ENABLED=0 cannot be used with that library as it requires C.
type Fuse struct {
	Cmd        []string `json:"cmd,omitempty"`
	Entrypoint []string `json:"entrypoint,omitempty"`
	Env        []string `json:"env,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
}

type InitSpec struct {
	Command    []string
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
		justMkdir := len(m.fsType) == 0
		if justMkdir {
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
			path.Join(dirRoot, "dev"),
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
		findPath := path.Join(dirRoot, dir, executable)
		fi, err := os.Stat(findPath)
		if err != nil {
			continue
		}
		if fi.Mode()&execBits != 0 {
			return path.Join(dir, executable), nil
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

func metadataToInitSpec(metadata *v1.ConfigFile) (*InitSpec, error) {
	initSpec := &InitSpec{}

	var command []string
	if metadata.Config.Entrypoint != nil {
		command = metadata.Config.Entrypoint
	}
	if metadata.Config.Cmd != nil {
		command = append(command, metadata.Config.Cmd...)
	}
	if command == nil {
		command = []string{"/bin/sh"}
	}

	pathEnv := pathDefault
	pathMetadata := getenv(metadata.Config.Env, "PATH")
	if len(pathMetadata) > 0 {
		pathEnv = pathMetadata
	}

	if !strings.HasPrefix(command[0], dirRoot) {
		executablePath, err := findExecutableInPath(command[0], pathEnv)
		if err != nil {
			return nil, err
		}
		command[0] = executablePath
	}

	initSpec.Command = command

	initSpec.Env = metadata.Config.Env
	if initSpec.Env == nil {
		initSpec.Env = os.Environ()
	}

	initSpec.WorkingDir = metadata.Config.WorkingDir
	if len(initSpec.WorkingDir) == 0 {
		initSpec.WorkingDir = dirRoot
	}

	return initSpec, nil
}

func getUserGroup(userEntry string) (int, int, error) {
	var (
		user  string
		group string
		uid   int
		gid   int
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

	p, err := os.Open(filePasswd)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open %s: %w", filePasswd, err)
	}
	defer p.Close()

	userFound := false
	pScanner := bufio.NewScanner(p)
	for pScanner.Scan() {
		line := pScanner.Text()
		fields := strings.Split(line, ":")
		if fields[0] == user {
			userFound = true
			uid, err = strconv.Atoi(fields[2])
			if err != nil {
				return 0, 0, fmt.Errorf("unexpected error reading %s: %w", filePasswd, err)
			}
			break
		}
	}
	if err = pScanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("unable to read %s: %w", filePasswd, err)
	}
	if !userFound {
		return 0, 0, fmt.Errorf("user %s not found", user)
	}

	if len(group) == 0 || group == "root" {
		return uid, gid, nil
	}

	g, err := os.Open(fileGroup)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open %s: %w", fileGroup, err)
	}
	defer g.Close()

	groupFound := false
	gScanner := bufio.NewScanner(g)
	for gScanner.Scan() {
		line := gScanner.Text()
		fields := strings.Split(line, ":")
		if fields[0] == group {
			groupFound = true
			gid, err = strconv.Atoi(fields[2])
			if err != nil {
				return 0, 0, fmt.Errorf("unexpected error reading %s: %w", fileGroup, err)
			}
			break
		}
	}
	if err = gScanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("unable to read %s: %w", fileGroup, err)
	}
	if !groupFound {
		return 0, 0, fmt.Errorf("group %s not found", group)
	}

	return uid, gid, nil
}

func DoIt() error {
	fmt.Println("Starting init")

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

	debug()
	fmt.Println("After debug()")

	metadata, err := readMetadata(filepath.Join(dirCB, fileMetadata))
	if err != nil {
		return err
	}
	initSpec, err := metadataToInitSpec(metadata)
	if err != nil {
		return err
	}

	if len(metadata.Config.WorkingDir) != 0 {
		err = os.Chdir(metadata.Config.WorkingDir)
		if err != nil {
			return fmt.Errorf("unable to change working directory to %s: %w",
				metadata.Config.WorkingDir, err)
		}
	}

	initSpec.UID, initSpec.GID, err = getUserGroup(metadata.Config.User)
	if err != nil {
		return err
	}

	err = syscall.Setuid(initSpec.UID)
	if err != nil {
		return fmt.Errorf("unable to set UID: %w", err)
	}

	err = syscall.Setgid(initSpec.GID)
	if err != nil {
		return fmt.Errorf("unable to set GID: %w", err)
	}

	return syscall.Exec(initSpec.Command[0], initSpec.Command, initSpec.Env)
}
