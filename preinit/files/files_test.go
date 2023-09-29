package files

import (
	"fmt"
	"testing"

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

func ExampleDescendingDirs() {
	fmt.Println(DescendingDirs("abc/123/000/343"))
	// Output: [abc abc/123 abc/123/000 abc/123/000/343]
}
