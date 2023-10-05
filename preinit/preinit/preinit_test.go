package preinit

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/cloudboss/easyto/preinit/aws"
	"github.com/cloudboss/easyto/preinit/maps"
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
		env     NameValueSource
		envFrom EnvFromSource
		result  NameValueSource
		err     error
		fail    bool
	}{
		{
			env:     NameValueSource{},
			envFrom: EnvFromSource{},
			result:  NameValueSource{},
			err:     nil,
		},
		{
			env: NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			envFrom: EnvFromSource{},
			result: NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			err: nil,
		},
		{
			env: NameValueSource{},
			envFrom: EnvFromSource{
				{
					SSMParameter: &SSMParameterEnvSource{
						Path: "/aaaaa",
					},
				},
			},
			result: NameValueSource{
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
			env: NameValueSource{
				{
					Name:  "CDE",
					Value: "cde",
				},
			},
			envFrom: EnvFromSource{
				{
					SSMParameter: &SSMParameterEnvSource{
						Path: "/aaaaa",
					},
				},
			},
			result: NameValueSource{
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
			env: NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			envFrom: EnvFromSource{
				{
					SSMParameter: &SSMParameterEnvSource{
						Path: "/aaaaa",
					},
				},
			},
			result: NameValueSource{
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
			env: NameValueSource{},
			envFrom: EnvFromSource{
				{
					SSMParameter: &SSMParameterEnvSource{
						Path:     "/aaaaa",
						Optional: true,
					},
				},
			},
			result: NameValueSource{},
			err:    nil,
			fail:   true,
		},
		{
			env: NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			envFrom: EnvFromSource{
				{
					SSMParameter: &SSMParameterEnvSource{
						Path:     "/aaaaa",
						Optional: true,
					},
				},
			},
			result: NameValueSource{
				{
					Name:  "ABC",
					Value: "abc",
				},
			},
			err:  nil,
			fail: true,
		},
		{
			env: NameValueSource{},
			envFrom: EnvFromSource{
				{
					SSMParameter: &SSMParameterEnvSource{
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
