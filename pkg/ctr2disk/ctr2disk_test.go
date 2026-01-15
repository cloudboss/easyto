package ctr2disk

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudboss/easyto/pkg/testutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test pure utility functions

func TestPartitionName(t *testing.T) {
	testCases := []struct {
		description string
		disk        string
		partition   int
		expected    string
	}{
		{
			description: "Standard disk naming",
			disk:        "/dev/sda",
			partition:   1,
			expected:    "/dev/sda1",
		},
		{
			description: "Numbered disk naming",
			disk:        "/dev/nvme0n1",
			partition:   1,
			expected:    "/dev/nvme0n1p1",
		},
		{
			description: "Loop device",
			disk:        "/dev/loop0",
			partition:   2,
			expected:    "/dev/loop0p2",
		},
		{
			description: "SCSI disk",
			disk:        "/dev/sdb",
			partition:   5,
			expected:    "/dev/sdb5",
		},
		{
			description: "MMC block device",
			disk:        "/dev/mmcblk0",
			partition:   1,
			expected:    "/dev/mmcblk0p1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := partitionName(tc.disk, tc.partition)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatBootEntry(t *testing.T) {
	b := &Builder{
		kernelVersion: "6.12.63",
		uuidRoot:      "12345678-1234-1234-1234-123456789abc",
	}

	entry := b.formatBootEntry(b.uuidRoot)

	// Verify it contains expected components
	assert.Contains(t, entry, "linux /vmlinuz-6.12.63")
	assert.Contains(t, entry, "root=PARTUUID=12345678-1234-1234-1234-123456789abc")
	assert.Contains(t, entry, "console=tty0")
	assert.Contains(t, entry, "console=ttyS0,115200")
	assert.Contains(t, entry, "init=")
	assert.Contains(t, entry, "/sbin/init")
	assert.Contains(t, entry, "rw")

	// Verify format is correct
	lines := bytes.Split([]byte(entry), []byte("\n"))
	assert.GreaterOrEqual(t, len(lines), 2)
	assert.Contains(t, string(lines[0]), "linux")
	assert.Contains(t, string(lines[1]), "options")
}

func TestNewErrExtract(t *testing.T) {
	testCases := []struct {
		description string
		code        byte
		wrapErr     error
		expected    string
	}{
		{
			description: "Block device error",
			code:        tar.TypeBlock,
			wrapErr:     os.ErrPermission,
			expected:    "unable to create block device",
		},
		{
			description: "Character device error",
			code:        tar.TypeChar,
			wrapErr:     os.ErrPermission,
			expected:    "unable to create character device",
		},
		{
			description: "Directory error",
			code:        tar.TypeDir,
			wrapErr:     os.ErrPermission,
			expected:    "unable to create directory",
		},
		{
			description: "FIFO error",
			code:        tar.TypeFifo,
			wrapErr:     os.ErrPermission,
			expected:    "unable to create fifo",
		},
		{
			description: "Hard link error",
			code:        tar.TypeLink,
			wrapErr:     os.ErrPermission,
			expected:    "unable to create hard link",
		},
		{
			description: "Regular file error",
			code:        tar.TypeReg,
			wrapErr:     os.ErrPermission,
			expected:    "unable to create file",
		},
		{
			description: "Symbolic link error",
			code:        tar.TypeSymlink,
			wrapErr:     os.ErrPermission,
			expected:    "unable to create symbolic link",
		},
		{
			description: "Mode error",
			code:        'Y',
			wrapErr:     os.ErrPermission,
			expected:    "unable to set permissions",
		},
		{
			description: "Timestamp error",
			code:        'Z',
			wrapErr:     os.ErrPermission,
			expected:    "unable to set timestamp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := newErrExtract(rune(tc.code), tc.wrapErr)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "error extracting tar archive")
			assert.Contains(t, err.Error(), tc.expected)
			assert.Contains(t, err.Error(), tc.wrapErr.Error())
		})
	}
}

func TestKernelVersionFromArchive(t *testing.T) {
	testCases := []struct {
		description   string
		files         map[string]string
		expectedVer   string
		expectError   bool
		errorContains string
	}{
		{
			description: "Valid kernel archive",
			files: map[string]string{
				"./boot/vmlinuz-6.12.63": "fake kernel data",
				"./lib/modules/6.12.63/": "",
			},
			expectedVer: "6.12.63",
			expectError: false,
		},
		{
			description: "Different kernel version",
			files: map[string]string{
				"./boot/vmlinuz-5.15.0": "fake kernel data",
			},
			expectedVer: "5.15.0",
			expectError: false,
		},
		{
			description: "No kernel in archive",
			files: map[string]string{
				"./lib/modules/6.12.63/": "",
			},
			expectError:   true,
			errorContains: "unable to find kernel",
		},
		{
			description:   "Empty archive",
			files:         map[string]string{},
			expectError:   true,
			errorContains: "unable to find kernel",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			testFS := afero.NewMemMapFs()
			tarPath := "/tmp/kernel.tar"
			err := testutil.WriteTarFile(testFS, tarPath, tc.files)
			require.NoError(t, err)

			version, err := kernelVersionFromArchive(testFS, tarPath)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedVer, version)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	content := "test file content"
	src := bytes.NewBufferString(content)
	testFS := afero.NewMemMapFs()
	destPath := "/test/file.txt"

	err := testFS.MkdirAll("/test", 0755)
	require.NoError(t, err)

	err = copyFile(testFS, src, destPath, 0644)
	require.NoError(t, err)

	readContent, err := afero.ReadFile(testFS, destPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(readContent))

	info, err := testFS.Stat(destPath)
	require.NoError(t, err)
	assert.True(t, info.Mode().IsRegular())
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}

func TestUntarFile(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Test requires root privileges for lchown operations")
	}

	files := map[string]string{
		"file1.txt":     "content1",
		"dir/file2.txt": "content2",
	}

	tarData, err := testutil.CreateTarArchive(files)
	require.NoError(t, err)

	tmpTar, err := os.CreateTemp("", "test-*.tar")
	require.NoError(t, err)
	defer os.Remove(tmpTar.Name())
	_, err = tmpTar.Write(tarData)
	require.NoError(t, err)
	tmpTar.Close()

	destDir, err := os.MkdirTemp("", "untar-*")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	err = untarFile(afero.NewOsFs(), tmpTar.Name(), destDir)
	require.NoError(t, err)

	for path, expectedContent := range files {
		fullPath := filepath.Join(destDir, path)
		content, err := os.ReadFile(fullPath)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(content))
	}
}

func TestReadlink(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readlink-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	realFile := filepath.Join(tmpDir, "real.txt")
	err = os.WriteFile(realFile, []byte("content"), 0644)
	require.NoError(t, err)

	symlinkFile := filepath.Join(tmpDir, "link.txt")
	err = os.Symlink(realFile, symlinkFile)
	require.NoError(t, err)

	relativeLink := filepath.Join(tmpDir, "relative.txt")
	err = os.Symlink("real.txt", relativeLink)
	require.NoError(t, err)

	testCases := []struct {
		description string
		path        string
		expected    string
		expectError bool
	}{
		{
			description: "Regular file",
			path:        realFile,
			expected:    realFile,
			expectError: false,
		},
		{
			description: "Absolute symlink",
			path:        symlinkFile,
			expected:    realFile,
			expectError: false,
		},
		{
			description: "Relative symlink",
			path:        relativeLink,
			expected:    realFile,
			expectError: false,
		},
		{
			description: "Non-existent file",
			path:        filepath.Join(tmpDir, "nonexistent"),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result, err := readlink(tc.path)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// Test Builder options

func TestBuilderOptions(t *testing.T) {
	testCases := []struct {
		description string
		opts        []BuilderOpt
		verify      func(*testing.T, *Builder)
	}{
		{
			description: "WithAssetDir",
			opts:        []BuilderOpt{WithAssetDir("/test/assets")},
			verify: func(t *testing.T, b *Builder) {
				assert.Equal(t, "/test/assets", b.AssetDir)
			},
		},
		{
			description: "WithCTRImageName",
			opts:        []BuilderOpt{WithCTRImageName("test:latest")},
			verify: func(t *testing.T, b *Builder) {
				assert.Equal(t, "test:latest", b.CTRImageName)
			},
		},
		{
			description: "WithCTRImageSource",
			opts:        []BuilderOpt{WithCTRImageSource("daemon")},
			verify: func(t *testing.T, b *Builder) {
				assert.Equal(t, "daemon", b.CTRImageSource)
			},
		},
		{
			description: "WithVMImageDevice",
			opts:        []BuilderOpt{WithVMImageDevice("/dev/sda")},
			verify: func(t *testing.T, b *Builder) {
				assert.Equal(t, "/dev/sda", b.VMImageDevice)
			},
		},
		{
			description: "WithVMImageMount",
			opts:        []BuilderOpt{WithVMImageMount("/mnt")},
			verify: func(t *testing.T, b *Builder) {
				assert.Equal(t, "/mnt", b.VMImageMount)
			},
		},
		{
			description: "WithServices",
			opts:        []BuilderOpt{WithServices([]string{"chrony", "ssh"})},
			verify: func(t *testing.T, b *Builder) {
				assert.Equal(t, []string{"chrony", "ssh"}, b.Services)
			},
		},
		{
			description: "WithLoginUser",
			opts:        []BuilderOpt{WithLoginUser("testuser")},
			verify: func(t *testing.T, b *Builder) {
				assert.Equal(t, "testuser", b.LoginUser)
			},
		},
		{
			description: "WithLoginShell",
			opts:        []BuilderOpt{WithLoginShell("/bin/bash")},
			verify: func(t *testing.T, b *Builder) {
				assert.Equal(t, "/bin/bash", b.LoginShell)
			},
		},
		{
			description: "WithDebug",
			opts:        []BuilderOpt{WithDebug(true)},
			verify: func(t *testing.T, b *Builder) {
				assert.True(t, b.Debug)
			},
		},
		{
			description: "Multiple options",
			opts: []BuilderOpt{
				WithAssetDir("/assets"),
				WithCTRImageName("alpine:latest"),
				WithDebug(true),
			},
			verify: func(t *testing.T, b *Builder) {
				assert.Equal(t, "/assets", b.AssetDir)
				assert.Equal(t, "alpine:latest", b.CTRImageName)
				assert.True(t, b.Debug)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			b := &Builder{}
			for _, opt := range tc.opts {
				opt(b)
			}
			tc.verify(t, b)
		})
	}
}

func TestNewBuilder(t *testing.T) {
	testFS := afero.NewMemMapFs()
	tmpDir := "/tmp/builder-test"
	err := testFS.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	kernelFiles := map[string]string{
		"./boot/vmlinuz-6.12.63": "fake kernel",
	}
	kernelTar := filepath.Join(tmpDir, "kernel.tar")
	err = testutil.WriteTarFile(testFS, kernelTar, kernelFiles)
	require.NoError(t, err)

	testCases := []struct {
		description   string
		opts          []BuilderOpt
		expectError   bool
		errorContains string
	}{
		{
			description:   "Missing asset directory",
			opts:          []BuilderOpt{WithVMImageDevice("/dev/sda")},
			expectError:   true,
			errorContains: "asset directory must be defined",
		},
		{
			description:   "Missing VM image device",
			opts:          []BuilderOpt{WithAssetDir(tmpDir)},
			expectError:   true,
			errorContains: "VM image device must be defined",
		},
		{
			description: "Valid minimal builder",
			opts: []BuilderOpt{
				WithAssetDir(tmpDir),
				WithVMImageDevice("/dev/loop0"),
			},
			expectError: false,
		},
		{
			description: "Valid full builder",
			opts: []BuilderOpt{
				WithAssetDir(tmpDir),
				WithVMImageDevice("/dev/loop0"),
				WithCTRImageName("alpine:latest"),
				WithCTRImageSource("remote"),
				WithVMImageMount("/mnt"),
				WithServices([]string{"chrony"}),
				WithLoginUser("testuser"),
				WithLoginShell("/bin/sh"),
				WithDebug(true),
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			builder, err := NewBuilder(testFS, tc.opts...)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, builder)
				// Verify kernel version was extracted
				if !tc.expectError {
					assert.Equal(t, "6.12.63", builder.kernelVersion)
				}
			}
		})
	}
}

func TestLoadImage(t *testing.T) {
	t.Run("Unknown source", func(t *testing.T) {
		_, err := loadImage(nil, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown image source")
	})
}

func TestSetupServices(t *testing.T) {
	testCases := []struct {
		description   string
		services      []string
		expectError   bool
		errorContains string
	}{
		{
			description: "No services",
			services:    []string{},
			expectError: false,
		},
		{
			description:   "Unknown service",
			services:      []string{"unknown"},
			expectError:   true,
			errorContains: "unknown service",
		},
		{
			description:   "Unknown service first",
			services:      []string{"unknown", "chrony"},
			expectError:   true,
			errorContains: "unknown service",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			b := &Builder{
				Services: tc.services,
			}

			err := b.setupServices()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCopyFileErrors(t *testing.T) {
	t.Run("Read-only filesystem", func(t *testing.T) {
		testFS := afero.NewReadOnlyFs(afero.NewMemMapFs())
		src := bytes.NewBufferString("test content")
		destPath := "/test/file.txt"

		err := copyFile(testFS, src, destPath, 0644)
		assert.Error(t, err)
	})

	t.Run("Write with restricted permissions", func(t *testing.T) {
		testFS := afero.NewMemMapFs()
		destPath := "/test/file.txt"

		err := testFS.MkdirAll("/test", 0755)
		require.NoError(t, err)

		src := bytes.NewBufferString("test")
		err = copyFile(testFS, src, destPath, 0000)
		require.NoError(t, err)

		info, err := testFS.Stat(destPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0000), info.Mode().Perm())
	})
}

func TestSetupMetadata(t *testing.T) {
	t.Run("Write metadata successfully", func(t *testing.T) {
		config := &v1.ConfigFile{
			Config: v1.Config{
				Env: []string{"PATH=/usr/bin"},
				Cmd: []string{"/bin/sh"},
			},
		}
		img, err := testutil.CreateTestImage(config)
		require.NoError(t, err)

		tmpFile, err := os.CreateTemp("", "metadata-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		b := &Builder{}
		err = b.setupMetadata(img, tmpFile.Name())
		require.NoError(t, err)

		data, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(data), "PATH=/usr/bin")
		assert.Contains(t, string(data), "/bin/sh")
	})

	t.Run("Error when directory does not exist", func(t *testing.T) {
		config := &v1.ConfigFile{}
		img, err := testutil.CreateTestImage(config)
		require.NoError(t, err)

		b := &Builder{}
		err = b.setupMetadata(img, "/nonexistent/directory/metadata.json")
		assert.Error(t, err)
	})
}
