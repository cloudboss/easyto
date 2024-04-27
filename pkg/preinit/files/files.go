package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// DescendingDirs returns an array of directory names where each subsequent
// name in the array is one level deeper than the previous.
func DescendingDirs(dir string) []string {
	return descendingDirs(dir, "")
}

func descendingDirs(dir, acc string) []string {
	if len(dir) == 0 {
		return []string{}
	}
	dirs := strings.Split(dir, string(os.PathSeparator))
	if len(dirs[0]) == 0 {
		// dir is an absolute path.
		dirs[0] = string(os.PathSeparator)
	}
	newAcc := filepath.Join(acc, dirs[0])
	return append([]string{newAcc}, descendingDirs(filepath.Join(dirs[1:]...), newAcc)...)
}

func Mkdirs(fs afero.Fs, dir string, uid, gid int, mode os.FileMode) error {
	for _, d := range DescendingDirs(dir) {
		err := fs.Mkdir(d, mode)
		if !(err == nil || os.IsExist(err)) {
			return fmt.Errorf("unable to create directory %s: %w", d, err)
		}
		if os.IsExist(err) {
			continue
		}
		err = fs.Chown(d, uid, gid)
		if err != nil {
			return fmt.Errorf("unable to set permissions on directory %s: %w", d, err)
		}
	}
	return nil
}
