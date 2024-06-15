package maps

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func Test_ParameterMap_Write(t *testing.T) {
	testCases := []struct {
		pmap  ParameterMap
		dirs  []string
		files map[string][]byte
		err   error
	}{
		{
			pmap: ParameterMap{},
			err:  nil,
		},
		{
			pmap:  ParameterMap{"abc": "xyz"},
			dirs:  []string{"dest"},
			files: map[string][]byte{},
			err:   nil,
		},
		{
			pmap: ParameterMap{
				"xyz": ParameterMap{
					"xyz": "123",
				},
				"zzz": ParameterMap{
					"x": "0000",
					"y": ParameterMap{
						"a": "1111",
					},
				},
			},
			dirs: []string{"dest/xyz", "dest/zzz", "dest/zzz/y"},
			files: map[string][]byte{
				"dest/xyz/xyz": []byte("123"),
				"dest/zzz/x":   []byte("0000"),
				"dest/zzz/y/a": []byte("1111"),
			},
			err: nil,
		},
	}
	for _, tc := range testCases {
		fs := afero.NewMemMapFs()
		tc.pmap.SetFS(fs)

		err := tc.pmap.Write("dest", "", -1, -1)
		assert.Equal(t, tc.err, err)

		for _, dir := range tc.dirs {
			exists, err := afero.DirExists(fs, dir)
			assert.True(t, exists, "directory %s does not exist", dir)
			assert.Nil(t, err)
		}

		for k, v := range tc.files {
			hasBytes, err := afero.FileContainsBytes(fs, k, v)
			assert.True(t, hasBytes, "file %s does not contain expected contents", k)
			assert.Nil(t, err)
		}
	}
}

func Test_ParameterMap_ToMapString(t *testing.T) {
	testCases := []struct {
		anyMap ParameterMap
		result map[string]string
	}{
		{
			anyMap: ParameterMap{},
			result: map[string]string{},
		},
		{
			anyMap: ParameterMap{
				"subpath": ParameterMap{
					"abc": "subpath-abc-value",
				},
			},
			result: map[string]string{},
		},
		{
			anyMap: ParameterMap{
				"abc": "abc-value",
				"subpath": ParameterMap{
					"abc": "subpath-abc-value",
				},
				"xyz": "xyz-value",
			},
			result: map[string]string{
				"abc": "abc-value",
				"xyz": "xyz-value",
			},
		},
	}
	for _, tc := range testCases {
		actual := tc.anyMap.ToMapString()
		assert.EqualValues(t, tc.result, actual)
	}
}
