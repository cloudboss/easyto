package aws

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/afero"
)

func stringRC(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

type file struct {
	name    string
	content string
	mode    os.FileMode
}

func fileRead(fs afero.Fs, path string) (string, os.FileInfo, error) {
	stat, err := fs.Stat(path)
	if err != nil {
		return "", nil, err
	}
	b, err := afero.ReadFile(fs, path)
	if err != nil {
		return "", nil, err
	}
	return string(b), stat, nil
}
