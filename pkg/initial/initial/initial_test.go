package initial

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/cloudboss/easyto/pkg/initial/aws"
	"github.com/cloudboss/easyto/pkg/initial/collections"
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
	asmClient *mockASMClient
	s3Client  *mockS3Client
	ssmClient *mockSSMClient
}

func newMockConnection(fail bool) *mockConnection {
	return &mockConnection{
		&mockASMClient{fail},
		&mockS3Client{fail},
		&mockSSMClient{fail},
	}
}

type mockASMClient struct {
	fail bool
}

func (m *mockASMClient) GetSecretList(secretID string) (collections.WritableList, error) {
	return nil, nil
}

func (m *mockASMClient) GetSecretMap(secretID string) (map[string]string, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	mapp := map[string]string{
		"JKL": "jkl-value",
		"MNO": "mno-value",
	}
	return mapp, nil
}

func (m *mockASMClient) GetSecretValue(secretID string) ([]byte, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	b := []byte("Two before narrow not relied how except moment myself")
	return b, nil
}

func (c *mockConnection) ASMClient() aws.ASMClient {
	return c.asmClient
}

func (c *mockConnection) SSMClient() aws.SSMClient {
	return c.ssmClient
}

func (c *mockConnection) S3Client() aws.S3Client {
	return c.s3Client
}

type mockS3Client struct {
	fail bool
}

func (m *mockS3Client) GetObjectList(bucket, keyPrefix string) (collections.WritableList, error) {
	return nil, nil
}

func (m *mockS3Client) GetObjectMap(bucket, keyPrefix string) (map[string]string, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	mapp := map[string]string{
		"JKL": "jkl-value",
		"MNO": "mno-value",
	}
	return mapp, nil
}

func (m *mockS3Client) GetObjectValue(bucket, keyPrefix string) ([]byte, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	b := []byte("Had denoting properly jointure you occasion directly raillery")
	return b, nil
}

type mockSSMClient struct {
	fail bool
}

func (m *mockSSMClient) GetParameterList(ssmPath string) (collections.WritableList, error) {
	return nil, nil
}

func (m *mockSSMClient) GetParameterMap(ssmPath string) (map[string]string, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	mapp := map[string]string{
		"ABC": "abc-value",
		"XYZ": "xyz-value",
	}
	return mapp, nil
}

func (m *mockSSMClient) GetParameterValue(ssmPath string) ([]byte, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	b := []byte("Occasional middletons everything so to")
	return b, nil
}

func Test_resolveAllEnvs(t *testing.T) {
	testCases := []struct {
		description string
		env         vmspec.NameValueSource
		envFrom     vmspec.EnvFromSource
		result      vmspec.NameValueSource
		err         error
		fail        bool
	}{
		{
			description: "Null test case",
			env:         vmspec.NameValueSource{},
			envFrom:     vmspec.EnvFromSource{},
			result:      vmspec.NameValueSource{},
			err:         nil,
		},
		{
			description: "Single env without EnvFrom",
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
			description: "No env with single SSM EnvFrom",
			env:         vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{
				{
					SSM: &vmspec.SSMEnvSource{
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
			description: "Single env with single SSM EnvFrom",
			env: vmspec.NameValueSource{
				{
					Name:  "CDE",
					Value: "cde",
				},
			},
			envFrom: vmspec.EnvFromSource{
				{
					SSM: &vmspec.SSMEnvSource{
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
			description: "Single env and single SSM EnvFrom with duplicate",
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
					SSM: &vmspec.SSMEnvSource{
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
			description: "Failed optional SSM EnvFrom",
			env:         vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{
				{
					SSM: &vmspec.SSMEnvSource{
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
			description: "Single env and failed optional SSM EnvFrom",
			env: vmspec.NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			envFrom: vmspec.EnvFromSource{
				{
					SSM: &vmspec.SSMEnvSource{
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
			description: "Failed non-optional SSM EnvFrom",
			env:         vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{
				{
					SSM: &vmspec.SSMEnvSource{
						Path:     "/aaaaa",
						Optional: false,
					},
				},
			},
			result: nil,
			err:    errors.Join(errors.New("fail")),
			fail:   true,
		},
		{
			description: "Mixed SSM and S3 EnvFrom",
			env:         vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{
				{
					SSM: &vmspec.SSMEnvSource{
						Path:     "/aaaaa",
						Optional: true,
					},
				},
				{
					S3: &vmspec.S3EnvSource{
						Bucket: "thebucket",
						Key:    "/bbbbb",
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
				{
					Name:  "JKL",
					Value: "jkl-value",
				},
				{
					Name:  "MNO",
					Value: "mno-value",
				},
			},
			err: nil,
		},
		{
			description: "Raw SSM and S3 EnvFrom with Name defined",
			env:         vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{
				{
					S3: &vmspec.S3EnvSource{
						Bucket: "thebucket",
						Key:    "/aaaaa",
						Name:   "S3",
					},
				},
				{
					SSM: &vmspec.SSMEnvSource{
						Path: "/bbbbb",
						Name: "SSM",
					},
				},
			},
			result: vmspec.NameValueSource{
				{
					Name:  "S3",
					Value: "Had denoting properly jointure you occasion directly raillery",
				},
				{
					Name:  "SSM",
					Value: "Occasional middletons everything so to",
				},
			},
			err: nil,
		},
		{
			description: "Base64 encoded SSM and S3 EnvFrom with Name defined",
			env:         vmspec.NameValueSource{},
			envFrom: vmspec.EnvFromSource{
				{
					SecretsManager: &vmspec.SecretsManagerEnvSource{
						Base64Encode: true,
						SecretID:     "secret-id",
						Name:         "ASM",
					},
				},
				{
					S3: &vmspec.S3EnvSource{
						Base64Encode: true,
						Bucket:       "thebucket",
						Key:          "/aaaaa",
						Name:         "S3",
					},
				},
				{
					SSM: &vmspec.SSMEnvSource{
						Base64Encode: true,
						Path:         "/bbbbb",
						Name:         "SSM",
					},
				},
			},
			result: vmspec.NameValueSource{
				{
					Name:  "ASM",
					Value: "VHdvIGJlZm9yZSBuYXJyb3cgbm90IHJlbGllZCBob3cgZXhjZXB0IG1vbWVudCBteXNlbGY=",
				},
				{
					Name:  "S3",
					Value: "SGFkIGRlbm90aW5nIHByb3Blcmx5IGpvaW50dXJlIHlvdSBvY2Nhc2lvbiBkaXJlY3RseSByYWlsbGVyeQ==",
				},
				{
					Name:  "SSM",
					Value: "T2NjYXNpb25hbCBtaWRkbGV0b25zIGV2ZXJ5dGhpbmcgc28gdG8=",
				},
			},
			err: nil,
		},
		{
			description: "Expand variables within values",
			env: vmspec.NameValueSource{
				{
					Name:  "ENV",
					Value: "value",
				},
				{
					Name:  "EXPAND_ENV",
					Value: "$(ENV)",
				},
				{
					Name:  "ESCAPED",
					Value: "$$(ENV)",
				},
				{
					Name:  "NOT_FOUND",
					Value: "$(NOT_FOUND)",
				},
				{
					Name:  "NO_EXPAND_EXPANDED",
					Value: "$(EXPAND_ENV)",
				},
				{
					Name:  "EXPAND_ASM",
					Value: "$(ASM)",
				},
				{
					Name:  "EXPAND_S3",
					Value: "$(S3)",
				},
				{
					Name:  "EXPAND_SSM",
					Value: "$(SSM)",
				},
				{
					Name:  "EXPAND_MULTIPLE",
					Value: "ENV: $(ENV), ASM: $(ASM), S3: $(S3), SSM: $(SSM)",
				},
			},
			envFrom: vmspec.EnvFromSource{
				{
					SecretsManager: &vmspec.SecretsManagerEnvSource{
						Base64Encode: true,
						SecretID:     "secret-id",
						Name:         "ASM",
					},
				},
				{
					S3: &vmspec.S3EnvSource{
						Base64Encode: true,
						Bucket:       "thebucket",
						Key:          "/aaaaa",
						Name:         "S3",
					},
				},
				{
					SSM: &vmspec.SSMEnvSource{
						Base64Encode: true,
						Path:         "/bbbbb",
						Name:         "SSM",
					},
				},
			},
			result: vmspec.NameValueSource{
				{
					Name:  "ENV",
					Value: "value",
				},
				{
					Name:  "EXPAND_ENV",
					Value: "value",
				},
				{
					Name:  "ESCAPED",
					Value: "$(ENV)",
				},
				{
					Name:  "NOT_FOUND",
					Value: "$(NOT_FOUND)",
				},
				{
					Name:  "NO_EXPAND_EXPANDED",
					Value: "$(ENV)",
				},
				{
					Name:  "EXPAND_ASM",
					Value: "VHdvIGJlZm9yZSBuYXJyb3cgbm90IHJlbGllZCBob3cgZXhjZXB0IG1vbWVudCBteXNlbGY=",
				},
				{
					Name:  "EXPAND_S3",
					Value: "SGFkIGRlbm90aW5nIHByb3Blcmx5IGpvaW50dXJlIHlvdSBvY2Nhc2lvbiBkaXJlY3RseSByYWlsbGVyeQ==",
				},
				{
					Name:  "EXPAND_SSM",
					Value: "T2NjYXNpb25hbCBtaWRkbGV0b25zIGV2ZXJ5dGhpbmcgc28gdG8=",
				},
				{
					Name:  "EXPAND_MULTIPLE",
					Value: "ENV: value, ASM: VHdvIGJlZm9yZSBuYXJyb3cgbm90IHJlbGllZCBob3cgZXhjZXB0IG1vbWVudCBteXNlbGY=, S3: SGFkIGRlbm90aW5nIHByb3Blcmx5IGpvaW50dXJlIHlvdSBvY2Nhc2lvbiBkaXJlY3RseSByYWlsbGVyeQ==, SSM: T2NjYXNpb25hbCBtaWRkbGV0b25zIGV2ZXJ5dGhpbmcgc28gdG8=",
				},
				{
					Name:  "ASM",
					Value: "VHdvIGJlZm9yZSBuYXJyb3cgbm90IHJlbGllZCBob3cgZXhjZXB0IG1vbWVudCBteXNlbGY=",
				},
				{
					Name:  "S3",
					Value: "SGFkIGRlbm90aW5nIHByb3Blcmx5IGpvaW50dXJlIHlvdSBvY2Nhc2lvbiBkaXJlY3RseSByYWlsbGVyeQ==",
				},
				{
					Name:  "SSM",
					Value: "T2NjYXNpb25hbCBtaWRkbGV0b25zIGV2ZXJ5dGhpbmcgc28gdG8=",
				},
			},
			err: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			conn := newMockConnection(tc.fail)
			actual, err := resolveAllEnvs(conn, tc.env, tc.envFrom)
			assert.ElementsMatch(t, tc.result, actual)
			assert.EqualValues(t, tc.err, err)
		})
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
