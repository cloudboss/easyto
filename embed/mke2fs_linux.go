//go:build linux && amd64

package embed

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	//go:embed mke2fs.bin
	mke2fsBin []byte

	mke2fsOnce     sync.Once
	mke2fsExecPath string
	mke2fsInitErr  error
)

func mke2fsExtract() {
	dir, err := os.MkdirTemp("", "mke2fs-*")
	if err != nil {
		mke2fsInitErr = err
		return
	}

	path := filepath.Join(dir, "mke2fs")
	if err := os.WriteFile(path, mke2fsBin, 0o755); err != nil {
		mke2fsInitErr = err
		return
	}

	mke2fsExecPath = path
}

func mke2fs(args ...string) error {
	mke2fsOnce.Do(mke2fsExtract)
	if mke2fsInitErr != nil {
		return mke2fsInitErr
	}

	cmd := exec.Command(mke2fsExecPath, args...)
	return cmd.Run()
}

func MkfsExt4(device string, args ...string) error {
	mke2fsArgs := append([]string{"-t", "ext4"}, args...)
	mke2fsArgs = append(mke2fsArgs, device)
	return mke2fs(mke2fsArgs...)
}

func CleanupMke2fs() {
	os.RemoveAll(mke2fsExecPath)
}
