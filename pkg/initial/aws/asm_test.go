package aws

import (
	"testing"

	asm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/cloudboss/easyto/pkg/initial/maps"
	"github.com/stretchr/testify/assert"
)

func Test_secretToMap(t *testing.T) {
	testCases := []struct {
		description string
		name        string
		secret      asm.GetSecretValueOutput
		result      maps.ParameterMap
		err         error
	}{
		{
			description: "Null test case",
			secret:      asm.GetSecretValueOutput{},
			result:      nil,
			err:         ErrSecretNameRequired,
		},
		{
			description: "SecretString",
			name:        "name",
			secret: asm.GetSecretValueOutput{
				Name:         p("xyz"),
				SecretString: p("abc"),
			},
			result: maps.ParameterMap{
				"value": "abc",
			},
		},
		{
			description: "SecretBinary",
			name:        "name",
			secret: asm.GetSecretValueOutput{
				Name:         p("xyz"),
				SecretBinary: []byte("abc"),
			},
			result: maps.ParameterMap{
				"value": "abc",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actual, err := secretToMap(&tc.secret)
			assert.Equal(t, tc.err, err)
			assert.EqualValues(t, tc.result, actual)
		})
	}
}

func Test_secretMapToMap(t *testing.T) {
	testCases := []struct {
		description string
		secret      asm.GetSecretValueOutput
		result      maps.ParameterMap
		err         error
	}{
		{
			description: "Null test case",
			secret:      asm.GetSecretValueOutput{},
			result:      nil,
			err:         ErrSecretNameRequired,
		},
		{
			description: "SecretString",
			secret: asm.GetSecretValueOutput{
				Name:         p("xyz"),
				SecretString: p(`{"abc":"abc-value"}`),
			},
			result: maps.ParameterMap{
				"abc": "abc-value",
			},
		},
		{
			description: "Nested SecretString",
			secret: asm.GetSecretValueOutput{
				Name:         p("xyz"),
				SecretString: p(`{"abc":"abc-value","nested":{"def":"def-value"}}`),
			},
			result: maps.ParameterMap{
				"abc": "abc-value",
				"nested": map[string]any{
					"def": "def-value",
				},
			},
		},
		{
			description: "SecretBinary",
			secret: asm.GetSecretValueOutput{
				Name:         p("xyz"),
				SecretBinary: []byte(`{"abc":"abc-value"}`),
			},
			result: maps.ParameterMap{
				"abc": "abc-value",
			},
		},
		{
			description: "Nested SecretBinary",
			secret: asm.GetSecretValueOutput{
				Name:         p("xyz"),
				SecretBinary: []byte(`{"abc":"abc-value","nested":{"def":"def-value"}}`),
			},
			result: maps.ParameterMap{
				"abc": "abc-value",
				"nested": map[string]any{
					"def": "def-value",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actual, err := secretMapToMap(&tc.secret)
			assert.Equal(t, tc.err, err)
			assert.EqualValues(t, tc.result, actual)
		})
	}
}
