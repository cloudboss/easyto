package testutil

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTarArchive(t *testing.T) {
	files := map[string]string{
		"test.txt":     "hello world",
		"dir/file.txt": "nested file",
	}

	data, err := CreateTarArchive(files)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestWriteTarFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	files := map[string]string{
		"test.txt": "hello",
	}

	err := WriteTarFile(fs, "/archive.tar", files)
	require.NoError(t, err)

	exists, err := afero.Exists(fs, "/archive.tar")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCreateTestImage(t *testing.T) {
	config := &v1.ConfigFile{
		Config: v1.Config{
			Env: []string{"TEST=value"},
		},
	}

	img, err := CreateTestImage(config)
	require.NoError(t, err)
	assert.NotNil(t, img)

	configFile, err := img.ConfigFile()
	require.NoError(t, err)
	assert.Equal(t, []string{"TEST=value"}, configFile.Config.Env)
}
