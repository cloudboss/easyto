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
			description: "Overall merge",
			orig: &VMSpec{
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
			newVMSpec := tc.orig.Merge(tc.other)
			assert.Equal(t, tc.expected, newVMSpec)
		})
	}
}

func Test_NameValueSource_Merge(t *testing.T) {
	testCases := []struct {
		orig     NameValueSource
		other    NameValueSource
		expected NameValueSource
	}{
		{
			orig:     NameValueSource{},
			other:    NameValueSource{},
			expected: NameValueSource{},
		},
		{
			orig: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			other: nil,
			expected: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
		},
		{
			orig: nil,
			other: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			expected: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
		},
		{
			orig: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			other: NameValueSource{},
			expected: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
		},
		{
			orig: NameValueSource{},
			other: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			expected: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
		},
		{
			orig: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			other: NameValueSource{
				{
					Name:  "abc",
					Value: "yxz",
				},
			},
			expected: NameValueSource{
				{
					Name:  "abc",
					Value: "yxz",
				},
			},
		},
		{
			orig: NameValueSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
				{
					Name:  "xyz",
					Value: "xyz",
				},
			},
			other: NameValueSource{
				{
					Name:  "abc",
					Value: "yxz",
				},
			},
			expected: NameValueSource{
				{
					Name:  "abc",
					Value: "yxz",
				},
				{
					Name:  "xyz",
					Value: "xyz",
				},
			},
		},
	}
	for _, tc := range testCases {
		newEnvVars := tc.orig.Merge(tc.other)
		assert.ElementsMatch(t, tc.expected, newEnvVars)
	}
}
