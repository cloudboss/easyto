package preinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_VMSpec_userDataOverride(t *testing.T) {
	testCases := []struct {
		orig     VMSpec
		other    VMSpec
		expected VMSpec
	}{}
	for _, tc := range testCases {
		assert.Nil(t, tc)
	}
}

func Test_EnvVarSource_merge(t *testing.T) {
	testCases := []struct {
		orig     EnvVarSource
		other    EnvVarSource
		expected EnvVarSource
	}{
		{
			orig:     EnvVarSource{},
			other:    EnvVarSource{},
			expected: EnvVarSource{},
		},
		{
			orig: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			other: nil,
			expected: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
		},
		{
			orig: nil,
			other: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			expected: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
		},
		{
			orig: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			other: EnvVarSource{},
			expected: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
		},
		{
			orig: EnvVarSource{},
			other: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			expected: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
		},
		{
			orig: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
			},
			other: EnvVarSource{
				{
					Name:  "abc",
					Value: "yxz",
				},
			},
			expected: EnvVarSource{
				{
					Name:  "abc",
					Value: "yxz",
				},
			},
		},
		{
			orig: EnvVarSource{
				{
					Name:  "abc",
					Value: "xyz",
				},
				{
					Name:  "xyz",
					Value: "xyz",
				},
			},
			other: EnvVarSource{
				{
					Name:  "abc",
					Value: "yxz",
				},
			},
			expected: EnvVarSource{
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
		newEnvVars := tc.orig.merge(tc.other)
		assert.ElementsMatch(t, tc.expected, newEnvVars)
	}
}
