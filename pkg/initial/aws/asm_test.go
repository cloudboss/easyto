package aws

import (
	"context"
	"errors"
	"testing"

	asm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type mockASMAPI struct {
	secretBinary []byte
	secretString *string
}

func (m *mockASMAPI) GetSecretValue(ctx context.Context, in *asm.GetSecretValueInput,
	opt ...func(*asm.Options)) (*asm.GetSecretValueOutput, error) {
	out := &asm.GetSecretValueOutput{
		Name:         p("thesecret"),
		SecretBinary: m.secretBinary,
		SecretString: m.secretString,
	}
	return out, nil
}

func Test_ASMClient_GetSecretList(t *testing.T) {
	testCases := []struct {
		description  string
		dest         string
		secretString *string
		secretBinary []byte
		fsResult     []file
		err          error
	}{
		{
			description: "Null test case",
			fsResult:    []file{},
			err:         errors.New("secret thesecret has no value"),
		},
		{
			description:  "String secret",
			dest:         "/abc",
			secretString: p("abc-value"),
			fsResult: []file{
				{
					name:    "/abc",
					content: "abc-value",
					mode:    0600,
				},
			},
		},
		{
			description:  "Binary secret",
			dest:         "/def",
			secretBinary: []byte("def-value"),
			fsResult: []file{
				{
					name:    "/def",
					content: "def-value",
					mode:    0600,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			client := asmClient{
				api: &mockASMAPI{
					secretBinary: tc.secretBinary,
					secretString: tc.secretString,
				},
			}
			secrets, err := client.GetSecretList("thesecret")
			assert.Equal(t, tc.err, err)
			err = secrets.Write(fs, tc.dest, 0, 0, true)
			assert.NoError(t, err)
			for _, file := range tc.fsResult {
				contents, stat, err := fileRead(fs, file.name)
				assert.NoError(t, err)
				assert.Equal(t, string(file.content), contents)
				assert.Equal(t, file.mode, stat.Mode())
			}
		})
	}
}

func Test_ASMClient_GetSecretMap(t *testing.T) {
	testCases := []struct {
		description  string
		dest         string
		secretString *string
		secretBinary []byte
		result       map[string]string
		err          bool
	}{
		{
			description:  "Secret is not map",
			secretString: p("not-map"),
			err:          true,
		},
		{
			description:  "Secret is nested map",
			secretString: p(`{"abc": "123", "def": {"ghi": "789"}}`),
			err:          true,
		},
		{
			description:  "Secret is valid map",
			secretString: p(`{"abc": "123", "def": "456"}`),
			result: map[string]string{
				"abc": "123",
				"def": "456",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			client := asmClient{
				api: &mockASMAPI{
					secretBinary: tc.secretBinary,
					secretString: tc.secretString,
				},
			}
			secrets, err := client.GetSecretMap("thesecret")
			if tc.err {
				assert.Error(t, err)
			}
			assert.Equal(t, tc.result, secrets)
		})
	}
}

func Test_ASMClient_GetSecretValue(t *testing.T) {
	testCases := []struct {
		description  string
		dest         string
		secretString *string
		secretBinary []byte
		result       []byte
		err          error
	}{
		{
			description:  "Literal structured secret string value",
			secretString: p(`{"abc": "123", "def": "456"}`),
			result:       []byte(`{"abc": "123", "def": "456"}`),
		},
		{
			description:  "Literal unstructured secret string value",
			secretString: p("Ham him compass you proceed calling detract"),
			result:       []byte("Ham him compass you proceed calling detract"),
		},
		{
			description:  "Literal structured secret binary value",
			secretBinary: []byte(`{"abc": "123", "def": "456"}`),
			result:       []byte(`{"abc": "123", "def": "456"}`),
		},
		{
			description:  "Literal unstructured secret binary value",
			secretBinary: []byte("Ham him compass you proceed calling detract"),
			result:       []byte("Ham him compass you proceed calling detract"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			client := asmClient{
				api: &mockASMAPI{
					secretBinary: tc.secretBinary,
					secretString: tc.secretString,
				},
			}
			value, err := client.GetSecretValue("thesecret")
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.result, value)
		})
	}
}
