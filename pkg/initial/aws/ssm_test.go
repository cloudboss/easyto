package aws

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var (
	errParameterNotFound = errors.New("parameter not found")
)

type mockSSMAPI struct {
	parameters map[string]string
}

func (s *mockSSMAPI) GetParametersByPath(ctx context.Context, in *ssm.GetParametersByPathInput,
	opt ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	parameters := []types.Parameter{}
	for k := range s.parameters {
		if strings.HasPrefix(k, *in.Path) {
			parameters = append(parameters, types.Parameter{
				Name:  p(k),
				Value: p(s.parameters[k]),
			})
		}
	}
	out := &ssm.GetParametersByPathOutput{
		Parameters: parameters,
	}
	return out, nil
}

func (s *mockSSMAPI) GetParameter(ctx context.Context, in *ssm.GetParameterInput,
	opt ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	value, ok := s.parameters[*in.Name]
	if !ok {
		return nil, errParameterNotFound
	}
	out := &ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			Name:  in.Name,
			Value: &value,
		},
	}
	return out, nil
}

func Test_SSMClient_GetParameterList(t *testing.T) {
	testCases := []struct {
		description string
		dest        string
		parameters  map[string]string
		path        string
		result      []file
		secret      bool
		err         error
	}{
		{
			description: "Null test case",
			path:        "/zzzzz",
			err:         errParameterNotFound,
		},
		{
			description: "Nonsecret parameters",
			parameters: map[string]string{
				"/easy/to/abc":         "abc-value",
				"/easy/to/subpath/abc": "subpath-abc-value",
				"/easy/to/xyz":         "xyz-value",
			},
			dest: "/abc",
			path: "/easy/to",
			result: []file{
				{
					name:    "/abc/abc",
					content: "abc-value",
					mode:    0644,
				},
				{
					name: "/abc/subpath",
					mode: 0755 | os.ModeDir,
				},
				{
					name:    "/abc/subpath/abc",
					content: "subpath-abc-value",
					mode:    0644,
				},
				{
					name:    "/abc/xyz",
					content: "xyz-value",
					mode:    0644,
				},
			},
		},
		{
			description: "Secret parameters",
			parameters: map[string]string{
				"/easy/to/abc":         "abc-value",
				"/easy/to/subpath/abc": "subpath-abc-value",
				"/easy/to/xyz":         "xyz-value",
			},
			dest:   "/abc",
			path:   "/easy/to",
			secret: true,
			result: []file{
				{
					name:    "/abc/abc",
					content: "abc-value",
					mode:    0600,
				},
				{
					name: "/abc/subpath",
					mode: 0700 | os.ModeDir,
				},
				{
					name:    "/abc/subpath/abc",
					content: "subpath-abc-value",
					mode:    0600,
				},
				{
					name:    "/abc/xyz",
					content: "xyz-value",
					mode:    0600,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			client := ssmClient{
				api: &mockSSMAPI{tc.parameters},
			}
			parameters, err := client.GetParameterList(tc.path)
			assert.ErrorIs(t, err, tc.err)
			err = parameters.Write(fs, tc.dest, 0, 0, tc.secret)
			assert.NoError(t, err)
			for _, file := range tc.result {
				contents, stat, err := fileRead(fs, file.name)
				assert.NoError(t, err)
				assert.Equal(t, string(file.content), contents)
				assert.Equal(t, file.mode, stat.Mode())
			}
		})
	}
}

func Test_SSMClient_GetParameterMap(t *testing.T) {
	testCases := []struct {
		description string
		parameters  map[string]string
		path        string
		result      map[string]string
		err         bool
	}{
		{
			description: "Parameter not found",
			path:        "/zzzzz",
			err:         true,
		},
		{
			description: "Invalid parameter is not map",
			parameters: map[string]string{
				"/easy/to/abc":         "abc-value",
				"/easy/to/subpath/abc": "subpath-abc-value",
				"/easy/to/xyz":         `"abc": "123", "def": "456"}`,
			},
			path: "/easy/to/xyz",
			err:  true,
		},
		{
			description: "Valid parameter is map",
			parameters: map[string]string{
				"/easy/to/abc":         "abc-value",
				"/easy/to/subpath/abc": "subpath-abc-value",
				"/easy/to/xyz":         `{"abc": "123", "def": "456"}`,
			},
			path: "/easy/to/xyz",
			result: map[string]string{
				"abc": "123",
				"def": "456",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			client := ssmClient{
				api: &mockSSMAPI{tc.parameters},
			}
			parameters, err := client.GetParameterMap(tc.path)
			if tc.err {
				assert.Error(t, err)
			}
			assert.Equal(t, tc.result, parameters)
		})
	}
}

func Test_SSMClient_GetParameterValue(t *testing.T) {
	testCases := []struct {
		description string
		parameters  map[string]string
		path        string
		result      []byte
		err         error
	}{
		{
			description: "Parameter not found",
			parameters: map[string]string{
				"/easy/to/abc":         "abc-value",
				"/easy/to/subpath/abc": "subpath-abc-value",
				"/easy/to/xyz":         `{"abc": "123", "def": "456"}`,
			},
			path: "/easy/to/xyz/123",
			err:  errParameterNotFound,
		},
		{
			description: "Literal structured parameter value",
			parameters: map[string]string{
				"/easy/to/abc":         "abc-value",
				"/easy/to/subpath/abc": "subpath-abc-value",
				"/easy/to/xyz":         `{"abc": "123", "def": "456"}`,
			},
			path:   "/easy/to/xyz",
			result: []byte(`{"abc": "123", "def": "456"}`),
		},
		{
			description: "Literal unstructured parameter value",
			parameters: map[string]string{
				"/easy/to/abc":         "abc-value",
				"/easy/to/subpath/abc": "subpath-abc-value",
				"/easy/to/xyz":         `"abc": "123", "def": "456"}`,
			},
			path:   "/easy/to/subpath/abc",
			result: []byte("subpath-abc-value"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			client := ssmClient{
				api: &mockSSMAPI{
					parameters: tc.parameters,
				},
			}
			value, err := client.GetParameterValue(tc.path)
			assert.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.result, value)
		})
	}
}
