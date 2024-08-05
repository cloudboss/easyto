package vmspec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_VMSpec_Merge(t *testing.T) {
	testCases := []struct {
		description string
		orig        *VMSpec
		other       *VMSpec
		expected    *VMSpec
	}{
		{
			description: "Null test case",
			orig:        &VMSpec{},
			other:       &VMSpec{},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{

			description: "Debug enabled",
			orig:        &VMSpec{},
			other: &VMSpec{
				Debug: true,
			},
			expected: &VMSpec{
				Debug: true,
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "ReplaceInit enabled",
			orig:        &VMSpec{},
			other: &VMSpec{
				ReplaceInit: true,
			},
			expected: &VMSpec{
				ReplaceInit: true,
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "Args and command overridden",
			orig: &VMSpec{
				Args:    []string{"abc"},
				Command: []string{"/usr/bin/xyz"},
			},
			other: &VMSpec{
				Args:    []string{"xyz"},
				Command: []string{"/usr/bin/abc"},
			},
			expected: &VMSpec{
				Args:    []string{"xyz"},
				Command: []string{"/usr/bin/abc"},
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "Args removed if command overridden",
			orig: &VMSpec{
				Args:    []string{"abc"},
				Command: []string{"/usr/bin/xyz"},
			},
			other: &VMSpec{
				Command: []string{"/usr/bin/abc"},
			},
			expected: &VMSpec{
				Args:    nil,
				Command: []string{"/usr/bin/abc"},
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "Security merged",
			orig: &VMSpec{
				Security: SecurityContext{
					ReadonlyRootFS: true,
				},
			},
			other: &VMSpec{
				Security: SecurityContext{
					RunAsGroupID: p(1234),
					RunAsUserID:  p(1234),
					SSHD: SSHD{
						Enable: true,
					},
				},
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					ReadonlyRootFS: true,
					RunAsGroupID:   p(1234),
					RunAsUserID:    p(1234),
					SSHD: SSHD{
						Enable: true,
					},
				},
			},
		},
		{
			description: "Security overriding with zero values",
			orig: &VMSpec{
				Security: SecurityContext{
					ReadonlyRootFS: true,
					RunAsGroupID:   p(1234),
					RunAsUserID:    p(1234),
				},
			},
			other: &VMSpec{
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					ReadonlyRootFS: true,
					RunAsGroupID:   p(0),
					RunAsUserID:    p(0),
				},
			},
		},
		{

			description: "Override disabled services",
			orig: &VMSpec{
				DisableServices: []string{"chrony", "ssh"},
			},
			other: &VMSpec{
				DisableServices: []string{"ssh"},
			},
			expected: &VMSpec{
				DisableServices: []string{"ssh"},
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "Mount overriding with zero values",
			orig: &VMSpec{
				Volumes: Volumes{
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/abc",
								GroupID:     p(1234),
								UserID:      p(1234),
							},
						},
					},
				},
			},
			other: &VMSpec{
				Volumes: Volumes{
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/xyz",
								GroupID:     p(0),
								UserID:      p(0),
							},
						},
					},
				},
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Volumes: Volumes{
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/xyz",
								GroupID:     p(0),
								UserID:      p(0),
							},
						},
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "Mount ownership defaults to command user and group",
			orig: &VMSpec{
				Security: SecurityContext{
					RunAsGroupID: p(1234),
					RunAsUserID:  p(1234),
				},
				Volumes: Volumes{
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/abc",
							},
						},
					},
				},
			},
			other: &VMSpec{},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Volumes: Volumes{
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/abc",
								GroupID:     p(1234),
								UserID:      p(1234),
							},
						},
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(1234),
					RunAsUserID:  p(1234),
				},
			},
		},
		{
			description: "Mount ownership can be explicitly set",
			orig: &VMSpec{
				Security: SecurityContext{
					RunAsGroupID: p(1234),
					RunAsUserID:  p(1234),
				},
				Volumes: Volumes{
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/abc",
								GroupID:     p(4321),
								UserID:      p(4321),
							},
						},
					},
				},
			},
			other: &VMSpec{},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Volumes: Volumes{
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/abc",
								GroupID:     p(4321),
								UserID:      p(4321),
							},
						},
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(1234),
					RunAsUserID:  p(1234),
				},
			},
		},
		{
			description: "NameValue null test case",
			orig: &VMSpec{
				Env: NameValueSource{},
			},
			other: &VMSpec{
				Env: NameValueSource{},
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "NameValue overridden",
			orig: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "xyz",
					},
				},
			},
			other: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "yxz",
					},
				},
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "yxz",
					},
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "NameValue empty merged into original",
			orig: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "xyz",
					},
				},
			},
			other: &VMSpec{
				Env: NameValueSource{},
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "xyz",
					},
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "NameValue PATH exists so not merged",
			orig: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: "/bin:/usr/bin",
					},
				},
			},
			other: &VMSpec{
				Env: NameValueSource{},
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "PATH",
						Value: "/bin:/usr/bin",
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "NameValue original merged into empty",
			orig: &VMSpec{
				Env: NameValueSource{},
			},
			other: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "xyz",
					},
				},
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "xyz",
					},
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "NameValue overridden and appended",
			orig: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "xyz",
					},
					{
						Name:  "foo",
						Value: "bar",
					},
				},
			},
			other: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "yxz",
					},
					{
						Name:  "bar",
						Value: "foo",
					},
				},
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "yxz",
					},
					{
						Name:  "foo",
						Value: "bar",
					},
					{
						Name:  "bar",
						Value: "foo",
					},
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(0),
					RunAsUserID:  p(0),
				},
			},
		},
		{
			description: "Overall merge",
			orig: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "xyz",
					},
				},
				Security: SecurityContext{
					ReadonlyRootFS: true,
				},
				Volumes: Volumes{
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/secret",
							},
							Path: "/ssm/path",
						},
					},
				},
			},
			other: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "zyx",
					},
					{
						Name:  "xyz",
						Value: "123",
					},
				},
				Security: SecurityContext{
					RunAsGroupID: p(4321),
					RunAsUserID:  p(1234),
					SSHD: SSHD{
						Enable: true,
					},
				},
				Volumes: Volumes{
					{
						EBS: &EBSVolumeSource{
							Device: "/dev/sda1",
							FSType: "ext4",
						},
					},
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/secret",
							},
							Path: "/ssm/path",
						},
					},
				},
				WorkingDir: "/tmp",
			},
			expected: &VMSpec{
				Env: NameValueSource{
					{
						Name:  "abc",
						Value: "zyx",
					},
					{
						Name:  "xyz",
						Value: "123",
					},
					{
						Name:  "PATH",
						Value: pathEnvDefault,
					},
				},
				Security: SecurityContext{
					ReadonlyRootFS: true,
					RunAsGroupID:   p(4321),
					RunAsUserID:    p(1234),
					SSHD: SSHD{
						Enable: true,
					},
				},
				Volumes: Volumes{
					{
						EBS: &EBSVolumeSource{
							Device: "/dev/sda1",
							FSType: "ext4",
							Mount: Mount{
								GroupID: p(4321),
								UserID:  p(1234),
							},
						},
					},
					{
						SSM: &SSMVolumeSource{
							Mount: Mount{
								Destination: "/secret",
								GroupID:     p(4321),
								UserID:      p(1234),
							},
							Path: "/ssm/path",
						},
					},
				},
				WorkingDir: "/tmp",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := tc.orig.Merge(tc.other)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, tc.orig)
		})
	}
}

func Test_VMSpec_Validate(t *testing.T) {
	testCases := []struct {
		description string
		orig        *VMSpec
		other       *VMSpec
		errMsg      *string
	}{
		{
			description: "Null test case",
			orig:        &VMSpec{},
			other:       &VMSpec{},
		},
		{
			description: "IMDS name required",
			orig:        &VMSpec{},
			other: &VMSpec{
				EnvFrom: EnvFromSource{
					{
						IMDS: &IMDSEnvSource{},
					},
				},
			},
			errMsg: p("env-from: imds name is required"),
		},
		{
			description: "Multiple sources and errors",
			orig:        &VMSpec{},
			other: &VMSpec{
				EnvFrom: EnvFromSource{
					{
						IMDS: &IMDSEnvSource{},
						S3:   &S3EnvSource{},
					},
				},
			},
			errMsg: p("env-from: imds name is required\nexpected 1 environment source, got 2: imds, s3-object"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := tc.orig.Merge(tc.other)
			assert.NoError(t, err)
			err = tc.orig.Validate()
			if tc.errMsg != nil {
				assert.EqualError(t, err, *tc.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
