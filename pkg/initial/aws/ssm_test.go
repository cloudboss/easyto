package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/cloudboss/easyto/pkg/initial/maps"
	"github.com/stretchr/testify/assert"
)

func Test_parametersToMap(t *testing.T) {
	testCases := []struct {
		description string
		parameters  []types.Parameter
		result      maps.ParameterMap
		prefix      string
	}{
		{
			description: "Null test case",
			parameters:  []types.Parameter{},
			result:      maps.ParameterMap{},
			prefix:      "/zzzzz",
		},
		{
			description: "Prefix elided",
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
			result: maps.ParameterMap{
				"abc": "abc-value",
				"subpath": maps.ParameterMap{
					"abc": "subpath-abc-value",
				},
				"xyz": "xyz-value",
			},
			prefix: "/easy/to",
		},
		{
			description: "Prefix not found",
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
			result: maps.ParameterMap{},
			prefix: "zzzzz",
		},
		{
			description: "File and directory collision",
			parameters: []types.Parameter{
				{
					Name:  p("/easy/to/abc"),
					Value: p("abc-value"),
				},
				// Value is not included in map because there is
				// an occurrence of /easy/to with a nested subpath.
				{
					Name:  p("/easy/to"),
					Value: p("to-value"),
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
			result: maps.ParameterMap{
				"to": maps.ParameterMap{
					"abc": "abc-value",
					"subpath": maps.ParameterMap{
						"abc": "subpath-abc-value",
					},
					"xyz": "xyz-value",
				},
			},
			prefix: "/easy",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actual := parametersToMap(tc.parameters, tc.prefix)
			assert.EqualValues(t, tc.result, actual)
		})
	}
}
