package collections

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cloudboss/easyto/pkg/initial/files"
	"github.com/spf13/afero"
)

type WritableListEntry struct {
	Path  string
	Value io.ReadCloser
}

type WritableList []*WritableListEntry

func (w WritableList) Write(fs afero.Fs, dest string, uid, gid int, secret bool) error {
	for _, le := range w {
		err := le.Write(fs, dest, uid, gid, secret)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w WritableListEntry) Write(fs afero.Fs, dest string, uid, gid int, secret bool) error {
	modeDir := os.FileMode(0755)
	modeFile := os.FileMode(0644)
	if secret {
		modeDir = os.FileMode(0700)
		modeFile = os.FileMode(0600)
	}

	finalDest := filepath.Join(dest, w.Path)
	destDir := filepath.Dir(finalDest)
	err := files.Mkdirs(fs, destDir, uid, gid, modeDir)
	if err != nil {
		return err
	}

	f, err := fs.OpenFile(finalDest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, modeFile)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %w", finalDest, err)
	}
	defer f.Close()

	_, err = io.Copy(f, w.Value)
	if err != nil {
		return fmt.Errorf("unable to write %s: %w", finalDest, err)
	}

	err = fs.Chown(finalDest, uid, gid)
	if err != nil {
		return fmt.Errorf("unable to set permissions on file %s: %w",
			finalDest, err)
	}

	return nil
}
