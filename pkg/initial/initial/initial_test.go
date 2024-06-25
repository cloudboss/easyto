package initial

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/cloudboss/easyto/pkg/initial/aws"
	"github.com/cloudboss/easyto/pkg/initial/maps"
	"github.com/cloudboss/easyto/pkg/initial/vmspec"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func Test_getenv(t *testing.T) {
	testCases := []struct {
		env      []string
		envVar   string
		expected string
	}{
		{
			env:      []string{},
			envVar:   "",
			expected: "",
		},
		{
			env: []string{
				"HOME=/root",
				"PATH=/bin:/sbin",
			},
			envVar:   "PATH",
			expected: "/bin:/sbin",
		},
	}
	for _, tc := range testCases {
		ev := getenv(tc.env, tc.envVar)
		assert.Equal(t, tc.expected, ev)
	}
}

func Test_parseMode(t *testing.T) {
	testCases := []struct {
		mode   string
		result fs.FileMode
		err    error
	}{
		{
			mode:   "",
			result: 0755,
			err:    nil,
		},
		{
			mode:   "0755",
			result: 0755,
			err:    nil,
		},
		{
			mode:   "0644",
			result: 0644,
			err:    nil,
		},
		{
			mode:   "abc",
			result: 0,
			err:    errors.New("invalid mode abc"),
		},
		{
			mode:   "-1",
			result: 0,
			err:    errors.New("invalid mode -1"),
		},
		{
			mode:   "258",
			result: 0,
			err:    errors.New("invalid mode 258"),
		},
		{
			mode:   "1234567890",
			result: 0,
			err:    errors.New("invalid mode 1234567890"),
		},
	}
	for _, tc := range testCases {
		actual, err := parseMode(tc.mode)
		assert.Equal(t, tc.result, actual)
		assert.Equal(t, tc.err, err)
	}
}

type mockConnection struct {
	ssmClient *mockSSMClient
}

func newMockConnection(fail bool) *mockConnection {
	return &mockConnection{
		&mockSSMClient{fail},
	}
}

func (c *mockConnection) SSMClient() aws.SSMClient {
	return c.ssmClient
}

func (c *mockConnection) S3Client() aws.S3Client {
	return nil
}

type mockSSMClient struct {
	fail bool
}

func (s *mockSSMClient) GetParameters(ssmPath string) (maps.ParameterMap, error) {
	if s.fail {
		return nil, errors.New("fail")
	}
	pMap := maps.ParameterMap{
		"ABC": "abc-value",
		"XYZ": "xyz-value",
		"subpath": maps.ParameterMap{
			"ABC": "subpath-abc-value",
		},
	}
	return pMap, nil
}

func Test_resolveAllEnvs(t *testing.T) {
	testCases := []struct {
		env     vmspec.NameValueSource
		envFrom vmspec.EnvFromSource
		result  vmspec.NameValueSource
		err     error
		fail    bool
	}{
		{
			env:     vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{},
			result:  vmspec.NameValueSource{},
			err:     nil,
		},
		{
			env: vmspec.NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			envFrom: vmspec.EnvFromSource{},
			result: vmspec.NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			err: nil,
		},
		{
			env: vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{
				{
					SSMParameter: &vmspec.SSMParameterEnvSource{
						Path: "/aaaaa",
					},
				},
			},
			result: vmspec.NameValueSource{
				{
					Name:  "ABC",
					Value: "abc-value",
				},
				{
					Name:  "XYZ",
					Value: "xyz-value",
				},
			},
			err: nil,
		},
		{
			env: vmspec.NameValueSource{
				{
					Name:  "CDE",
					Value: "cde",
				},
			},
			envFrom: vmspec.EnvFromSource{
				{
					SSMParameter: &vmspec.SSMParameterEnvSource{
						Path: "/aaaaa",
					},
				},
			},
			result: vmspec.NameValueSource{
				{
					Name:  "CDE",
					Value: "cde",
				},
				{
					Name:  "ABC",
					Value: "abc-value",
				},
				{
					Name:  "XYZ",
					Value: "xyz-value",
				},
			},
			err: nil,
		},
		{
			// Environment variable names within the image metadata are overridden
			// if they are defined in user data, but no check is done to ensure
			// there are no duplicates in the user data itself. Let execve() be the
			// decider on the behavior in this case.
			env: vmspec.NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			envFrom: vmspec.EnvFromSource{
				{
					SSMParameter: &vmspec.SSMParameterEnvSource{
						Path: "/aaaaa",
					},
				},
			},
			result: vmspec.NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
				{
					Name:  "ABC",
					Value: "abc-value",
				},
				{
					Name:  "XYZ",
					Value: "xyz-value",
				},
			},
			err: nil,
		},
		{
			env: vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{
				{
					SSMParameter: &vmspec.SSMParameterEnvSource{
						Path:     "/aaaaa",
						Optional: true,
					},
				},
			},
			result: vmspec.NameValueSource{},
			err:    nil,
			fail:   true,
		},
		{
			env: vmspec.NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			envFrom: vmspec.EnvFromSource{
				{
					SSMParameter: &vmspec.SSMParameterEnvSource{
						Path:     "/aaaaa",
						Optional: true,
					},
				},
			},
			result: vmspec.NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			err:  nil,
			fail: true,
		},
		{
			env: vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{
				{
					SSMParameter: &vmspec.SSMParameterEnvSource{
						Path:     "/aaaaa",
						Optional: false,
					},
				},
			},
			result: nil,
			err:    errors.Join(errors.New("fail")),
			fail:   true,
		},
	}
	for _, tc := range testCases {
		conn := newMockConnection(tc.fail)
		actual, err := resolveAllEnvs(conn, tc.env, tc.envFrom)
		assert.ElementsMatch(t, tc.result, actual)
		assert.EqualValues(t, tc.err, err)
	}
}

func Test_isMounted(t *testing.T) {
	const mtabPath = constants.DirProc + "/mounts"
	testCases := []struct {
		name         string
		mountPoint   string
		mtabPath     string
		mtabContents string
		mounted      bool
		errored      bool
	}{
		{
			name:       "returns error",
			mountPoint: "/abc",
			mtabPath:   "/wrong/mounts",
			mtabContents: `/dev/nvme0n1p2 /boot ext4 rw,seclabel,relatime 0 0
/dev/nvme0n1p1 /boot/efi vfat rw,relatime,fmask=0077,dmask=0077,codepage=437,iocharset=ascii,shortname=winnt,errors=remount-ro 0 0
tmpfs /tmp tmpfs rw,seclabel,nosuid,nodev,nr_inodes=1048576,inode64 0 0
binfmt_misc /proc/sys/fs/binfmt_misc binfmt_misc rw,nosuid,nodev,noexec,relatime 0 0`,
			mounted: false,
			errored: true,
		},
		{
			name:       "not mounted",
			mountPoint: "/abc",
			mtabPath:   mtabPath,
			mtabContents: `/dev/nvme0n1p2 /boot ext4 rw,seclabel,relatime 0 0
/dev/nvme0n1p1 /boot/efi vfat rw,relatime,fmask=0077,dmask=0077,codepage=437,iocharset=ascii,shortname=winnt,errors=remount-ro 0 0
tmpfs /tmp tmpfs rw,seclabel,nosuid,nodev,nr_inodes=1048576,inode64 0 0
binfmt_misc /proc/sys/fs/binfmt_misc binfmt_misc rw,nosuid,nodev,noexec,relatime 0 0`,
			mounted: false,
		},
		{
			name:       "is mounted",
			mountPoint: "/abc",
			mtabPath:   mtabPath,
			mtabContents: `/dev/nvme0n1p2 /boot ext4 rw,seclabel,relatime 0 0
/dev/nvme0n1p3 /abc ext4 rw,seclabel,relatime 0 0
/dev/nvme0n1p1 /boot/efi vfat rw,relatime,fmask=0077,dmask=0077,codepage=437,iocharset=ascii,shortname=winnt,errors=remount-ro 0 0
tmpfs /tmp tmpfs rw,seclabel,nosuid,nodev,nr_inodes=1048576,inode64 0 0
binfmt_misc /proc/sys/fs/binfmt_misc binfmt_misc rw,nosuid,nodev,noexec,relatime 0 0`,
			mounted: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			mounts, err := fs.OpenFile(mtabPath, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				t.Fatal(err)
			}
			defer mounts.Close()
			_, err = mounts.WriteString(tc.mtabContents)
			if err != nil {
				t.Fatal(err)
			}
			mounted, err := isMounted(fs, tc.mountPoint, tc.mtabPath)
			assert.Equal(t, tc.mounted, mounted)
			if err != nil {
				assert.True(t, tc.errored)
			}
		})
	}
}
