package testutil

import (
	"archive/tar"
	"bytes"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/spf13/afero"
)

// CreateTarArchive creates a tar archive with the given files.
// files is a map of path -> content.
func CreateTarArchive(files map[string]string) ([]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for path, content := range files {
		hdr := &tar.Header{
			Name: path,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WriteTarFile writes a tar archive to a file.
func WriteTarFile(fs afero.Fs, path string, files map[string]string) error {
	data, err := CreateTarArchive(files)
	if err != nil {
		return err
	}
	return afero.WriteFile(fs, path, data, 0644)
}

// CreateTestImage creates a minimal test container image with the given config.
func CreateTestImage(config *v1.ConfigFile) (v1.Image, error) {
	img := empty.Image

	if config != nil {
		var err error
		img, err = mutate.ConfigFile(img, config)
		if err != nil {
			return nil, err
		}
	}

	return img, nil
}
