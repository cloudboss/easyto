package maps

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudboss/easyto/preinit/files"
	"github.com/spf13/afero"
)

var fs = afero.NewOsFs()

type ParameterMap map[string]any

func (p ParameterMap) SetFS(newFS afero.Fs) {
	fs = newFS
}

func (p ParameterMap) Write(dest, subPath string, uid, gid int) error {
	var (
		source any
		ok     bool
	)

	source = p
	if len(subPath) > 0 {
		if source, ok = p[subPath]; !ok {
			return fmt.Errorf("subPath %s not found", subPath)
		}
	}

	switch value := source.(type) {
	// The value's type should be either a string or a nested ParameterMap.
	case string:
		return p.writeString(dest, value, uid, gid)
	case ParameterMap:
		return value.write(dest, uid, gid)
	}

	return nil
}

func (p ParameterMap) write(dest string, uid, gid int) error {
	for k, v := range p {
		newDest := filepath.Join(dest, k)
		switch value := v.(type) {
		case string:
			err := p.writeString(newDest, value, uid, gid)
			if err != nil {
				return fmt.Errorf("unable to write %s: %w", dest, err)
			}
		case ParameterMap:
			err := value.write(newDest, uid, gid)
			if err != nil {
				return fmt.Errorf("unable to write %s: %w", dest, err)
			}
		}
	}
	return nil
}

func (p ParameterMap) writeString(dest, value string, uid, gid int) error {
	const (
		modeDir  = 0700
		modeFile = 0600
	)

	destDir := filepath.Dir(dest)
	err := files.Mkdirs(fs, destDir, uid, gid, modeDir)
	if err != nil {
		return err
	}

	f, err := fs.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, modeFile)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %w", dest, err)
	}

	_, err = f.Write([]byte(value))
	if err != nil {
		return fmt.Errorf("unable to write %s: %w", dest, err)
	}

	err = fs.Chown(dest, uid, gid)
	if err != nil {
		return fmt.Errorf("unable to set permissions on file %s: %w",
			dest, err)
	}

	return nil
}

func (p ParameterMap) ToMapString() map[string]string {
	stringMap := map[string]string{}
	for k, v := range p {
		switch value := v.(type) {
		case string:
			stringMap[k] = value
		}
	}
	return stringMap
}
