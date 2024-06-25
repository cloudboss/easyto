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
				Security: SecurityContext{
					ReadonlyRootFS: true,
					RunAsGroupID:   p(0),
					RunAsUserID:    p(0),
				},
			},
		},
		{
			description: "Mount overriding with zero values",
			orig: &VMSpec{
				Volumes: Volumes{
					{
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/abc",
								GroupID:   p(1234),
								UserID:    p(1234),
							},
						},
					},
				},
			},
			other: &VMSpec{
				Volumes: Volumes{
					{
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/xyz",
								GroupID:   p(0),
								UserID:    p(0),
							},
						},
					},
				},
			},
			expected: &VMSpec{
				Volumes: Volumes{
					{
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/xyz",
								GroupID:   p(0),
								UserID:    p(0),
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
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/abc",
							},
						},
					},
				},
			},
			other: &VMSpec{},
			expected: &VMSpec{
				Volumes: Volumes{
					{
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/abc",
								GroupID:   p(1234),
								UserID:    p(1234),
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
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/abc",
								GroupID:   p(4321),
								UserID:    p(4321),
							},
						},
					},
				},
			},
			other: &VMSpec{},
			expected: &VMSpec{
				Volumes: Volumes{
					{
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/abc",
								GroupID:   p(4321),
								UserID:    p(4321),
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
				Env: NameValueSource{},
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
						SSMParameter: &SSMParameterVolumeSource{
							Mount: Mount{
								Directory: "/secret",
								GroupID:   p(4321),
								UserID:    p(1234),
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
