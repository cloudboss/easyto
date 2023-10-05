package files

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func Test_DescendingDirs(t *testing.T) {
	testCases := []struct {
		dir    string
		result []string
	}{
		{
			dir:    "",
			result: []string{},
		},
		{
			dir:    "abc",
			result: []string{"abc"},
		},
		{
			dir:    "abc/xyz",
			result: []string{"abc", "abc/xyz"},
		},
		{
			dir:    "abc/xyz/zzz",
			result: []string{"abc", "abc/xyz", "abc/xyz/zzz"},
		},
		{
			dir:    "abc///xyz/////zzz",
			result: []string{"abc", "abc/xyz", "abc/xyz/zzz"},
		},
		{
			dir:    "/",
			result: []string{"/"},
		},
		{
			dir:    "////",
			result: []string{"/"},
		},
		{
			dir:    "/abc/xyz/zzz",
			result: []string{"/", "/abc", "/abc/xyz", "/abc/xyz/zzz"},
		},
		{
			dir:    "/abc/////xyz///zzz",
			result: []string{"/", "/abc", "/abc/xyz", "/abc/xyz/zzz"},
		},
	}
	for _, tc := range testCases {
		actual := DescendingDirs(tc.dir)
		assert.ElementsMatch(t, tc.result, actual)
	}
}

func Test_Mkdirs(t *testing.T) {
	testCases := []struct {
		dir    string
		err    error
		result []string
	}{
		{
			dir:    "",
			err:    nil,
			result: []string{},
		},
		{
			dir:    "/",
			err:    nil,
			result: []string{"/"},
		},
		{
			dir:    "//",
			err:    nil,
			result: []string{"/"},
		},
		{
			dir: "/aaa/zzz/bbb/yyy",
			err: nil,
			result: []string{
				"/",
				"/aaa",
				"/aaa/zzz",
				"/aaa/zzz/bbb",
				"/aaa/zzz/bbb/yyy",
			},
		},
		{
			dir: "/aaa///zzz//bbb/////yyy",
			err: nil,
			result: []string{
				"/",
				"/aaa",
				"/aaa/zzz",
				"/aaa/zzz/bbb",
				"/aaa/zzz/bbb/yyy",
			},
		},
	}
	for _, tc := range testCases {
		fs := afero.NewMemMapFs()
		err := Mkdirs(fs, tc.dir, 0, 0, 755)
		assert.Equal(t, tc.err, err)
		for _, dir := range tc.result {
			t.Run(fmt.Sprintf("directory %s", dir), func(t *testing.T) {
				dirExists, err := afero.DirExists(fs, dir)
				assert.Nil(t, err, "error was not nil: %s", err)
				assert.True(t, dirExists, "directory %s does not exist", dir)
			})
		}
	}
}

func ExampleDescendingDirs() {
	fmt.Println(DescendingDirs("abc/123/000/343"))
	// Output: [abc abc/123 abc/123/000 abc/123/000/343]
}
