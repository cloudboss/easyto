package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/stretchr/testify/assert"
)

func Test_parametersToMap(t *testing.T) {
	testCases := []struct {
		parameters []types.Parameter
		result     map[string]any
		prefix     string
	}{
		{
			parameters: []types.Parameter{},
			result:     map[string]any{},
			prefix:     "/zzzzz",
		},
		{
			parameters: []types.Parameter{
				{
					Name:  p("/easy/to/abc"),
					Value: p("abc-value"),
				},
				{
					Name:  p("/easy/to/subpath/abc"),
					Value: p("subpath-abc-value"),
				},
				{
					Name:  p("/easy/to/xyz"),
					Value: p("xyz-value"),
				},
			},
			result: map[string]any{
				"abc": "abc-value",
				"subpath": map[string]any{
					"abc": "subpath-abc-value",
				},
				"xyz": "xyz-value",
			},
			prefix: "/easy/to",
		},
		{
			parameters: []types.Parameter{
				{
					Name:  p("/easy/to/abc"),
					Value: p("abc-value"),
				},
				{
					Name:  p("/easy/to/subpath/abc"),
					Value: p("subpath-abc-value"),
				},
				{
					Name:  p("/easy/to/xyz"),
					Value: p("xyz-value"),
				},
			},
			result: map[string]any{},
			prefix: "zzzzz",
		},
	}
	for _, tc := range testCases {
		actual := parametersToMap(tc.parameters, tc.prefix)
		assert.EqualValues(t, tc.result, actual)
	}
}
