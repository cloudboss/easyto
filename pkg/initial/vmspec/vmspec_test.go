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
			expected:    &VMSpec{},
		},
		{
			description: "Debug enabled",
			orig:        &VMSpec{},
			other: &VMSpec{
				Debug: true,
			},
			expected: &VMSpec{
				Debug: true,
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
					RunAsGroupID: 1234,
					RunAsUserID:  1234,
					SSHD: SSHD{
						Enable: true,
					},
				},
			},
			expected: &VMSpec{
				Security: SecurityContext{
					ReadonlyRootFS: true,
					RunAsGroupID:   1234,
					RunAsUserID:    1234,
					SSHD: SSHD{
						Enable: true,
					},
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
				Env: NameValueSource{},
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
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/secret",
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
					RunAsGroupID: 4321,
					RunAsUserID:  1234,
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
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/secret",
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
				},
				Security: SecurityContext{
					ReadonlyRootFS: true,
					RunAsGroupID:   4321,
					RunAsUserID:    1234,
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
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/secret",
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
