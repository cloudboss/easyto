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

func Test_NameValueSource_merge(t *testing.T) {
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
		newEnvVars := tc.orig.merge(tc.other)
		assert.ElementsMatch(t, tc.expected, newEnvVars)
	}
}
