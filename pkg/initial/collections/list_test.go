package collections

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func Test_WritableList_Write(t *testing.T) {
	testCases := []struct {
		description string
		dest        string
		in          WritableList
		secret      bool
		result      []file
		err         error
	}{
		{
			description: "Null test case",
			dest:        "dest",
			in:          nil,
		},
		{
			description: "Empty map",
			dest:        "dest",
			in:          WritableList{},
		},
		{
			description: "Single string entry",
			dest:        "/single",
			in: WritableList{
				&WritableListEntry{Path: "abc/xyz", Value: stringRC("xyz-1")},
			},
			result: []file{
				{
					name: "/single",
					mode: 0755 | os.ModeDir,
				},
				{
					name:    "/single/abc/xyz",
					content: "xyz-1",
					mode:    0644,
				},
			},
		},
		{
			description: "Multiple string entries",
			dest:        "/multiple",
			in: WritableList{
				&WritableListEntry{Path: "abc/def", Value: stringRC("def-1")},
				&WritableListEntry{Path: "ghi/jkl", Value: stringRC("jkl-1")},
			},
			result: []file{
				{
					name: "/multiple",
					mode: 0755 | os.ModeDir,
				},
				{
					name: "/multiple/abc",
					mode: 0755 | os.ModeDir,
				},
				{
					name:    "/multiple/abc/def",
					content: "def-1",
					mode:    0644,
				},
				{
					name:    "/multiple/ghi/jkl",
					content: "jkl-1",
					mode:    0644,
				},
			},
		},
		{
			description: "Nested string entries",
			dest:        "/nested",
			in: WritableList{
				&WritableListEntry{Path: "abc/def", Value: stringRC("def-1")},
				&WritableListEntry{Path: "abc/ghi/jkl", Value: stringRC("jkl-1")},
				&WritableListEntry{Path: "abc/ghi/mno", Value: stringRC("mno-1")},
				&WritableListEntry{Path: "abc/ghi/pqr/stu", Value: stringRC("stu-1")},
			},
			result: []file{
				{
					name: "/nested",
					mode: 0755 | os.ModeDir,
				},
				{
					name: "/nested/abc",
					mode: 0755 | os.ModeDir,
				},
				{
					name: "/nested/abc/ghi",
					mode: 0755 | os.ModeDir,
				},
				{
					name: "/nested/abc/ghi/pqr",
					mode: 0755 | os.ModeDir,
				},
				{
					name:    "/nested/abc/def",
					content: "def-1",
					mode:    0644,
				},
				{
					name:    "/nested/abc/ghi/jkl",
					content: "jkl-1",
					mode:    0644,
				},
				{
					name:    "/nested/abc/ghi/mno",
					content: "mno-1",
					mode:    0644,
				},
				{
					name:    "/nested/abc/ghi/pqr/stu",
					content: "stu-1",
					mode:    0644,
				},
			},
		},
		{
			description: "Single ReadCloser entry",
			dest:        "/single",
			in: WritableList{
				&WritableListEntry{Path: "abc/xyz", Value: stringRC("xyz-1")},
			},
			result: []file{
				{
					name: "/single",
					mode: 0755 | os.ModeDir,
				},
				{
					name:    "/single/abc/xyz",
					content: "xyz-1",
					mode:    0644,
				},
			},
		},
		{
			description: "Multiple ReadCloser entries",
			dest:        "/multiple",
			in: WritableList{
				&WritableListEntry{Path: "abc/def", Value: stringRC("def-1")},
				&WritableListEntry{Path: "ghi/jkl", Value: stringRC("jkl-1")},
			},
			result: []file{
				{
					name: "/multiple",
					mode: 0755 | os.ModeDir,
				},
				{
					name: "/multiple/abc",
					mode: 0755 | os.ModeDir,
				},
				{
					name:    "/multiple/abc/def",
					content: "def-1",
					mode:    0644,
				},
				{
					name:    "/multiple/ghi/jkl",
					content: "jkl-1",
					mode:    0644,
				},
			},
		},
		{
			description: "Nested mixed entries",
			dest:        "/nested",
			in: WritableList{
				&WritableListEntry{Path: "abc/def", Value: stringRC("def-1")},
				&WritableListEntry{Path: "abc/ghi/jkl", Value: stringRC("jkl-1")},
				&WritableListEntry{Path: "abc/ghi/mno", Value: stringRC("mno-1")},
				&WritableListEntry{Path: "abc/ghi/pqr/stu", Value: stringRC("stu-1")},
				&WritableListEntry{Path: "abc/ghi/pqr/vwx", Value: stringRC("vwx-1")},
			},
			result: []file{
				{
					name: "/nested",
					mode: 0755 | os.ModeDir,
				},
				{
					name: "/nested/abc",
					mode: 0755 | os.ModeDir,
				},
				{
					name: "/nested/abc/ghi",
					mode: 0755 | os.ModeDir,
				},
				{
					name: "/nested/abc/ghi/pqr",
					mode: 0755 | os.ModeDir,
				},
				{
					name:    "/nested/abc/def",
					content: "def-1",
					mode:    0644,
				},
				{
					name:    "/nested/abc/ghi/jkl",
					content: "jkl-1",
					mode:    0644,
				},
				{
					name:    "/nested/abc/ghi/mno",
					content: "mno-1",
					mode:    0644,
				},
				{
					name:    "/nested/abc/ghi/pqr/stu",
					content: "stu-1",
					mode:    0644,
				},
				{
					name:    "/nested/abc/ghi/pqr/vwx",
					content: "vwx-1",
					mode:    0644,
				},
			},
		},
		{
			description: "Nested mixed secret entries",
			dest:        "/nested",
			in: WritableList{
				&WritableListEntry{Path: "abc/def", Value: stringRC("def-1")},
				&WritableListEntry{Path: "abc/ghi/jkl", Value: stringRC("jkl-1")},
				&WritableListEntry{Path: "abc/ghi/mno", Value: stringRC("mno-1")},
				&WritableListEntry{Path: "abc/ghi/pqr/stu", Value: stringRC("stu-1")},
				&WritableListEntry{Path: "abc/ghi/pqr/vwx", Value: stringRC("vwx-1")},
			},
			secret: true,
			result: []file{
				{
					name: "/nested",
					mode: 0700 | os.ModeDir,
				},
				{
					name: "/nested/abc",
					mode: 0700 | os.ModeDir,
				},
				{
					name: "/nested/abc/ghi",
					mode: 0700 | os.ModeDir,
				},
				{
					name: "/nested/abc/ghi/pqr",
					mode: 0700 | os.ModeDir,
				},
				{
					name:    "/nested/abc/def",
					content: "def-1",
					mode:    0600,
				},
				{
					name:    "/nested/abc/ghi/jkl",
					content: "jkl-1",
					mode:    0600,
				},
				{
					name:    "/nested/abc/ghi/mno",
					content: "mno-1",
					mode:    0600,
				},
				{
					name:    "/nested/abc/ghi/pqr/stu",
					content: "stu-1",
					mode:    0600,
				},
				{
					name:    "/nested/abc/ghi/pqr/vwx",
					content: "vwx-1",
					mode:    0600,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			err := tc.in.Write(fs, tc.dest, 123, 456, tc.secret)
			assert.Equal(t, tc.err, err)
			for _, file := range tc.result {
				contents, stat, err := fileRead(fs, file.name)
				assert.NoError(t, err)
				assert.Equal(t, string(file.content), contents)
				assert.Equal(t, file.mode, stat.Mode())
			}
		})
	}
}

type file struct {
	name    string
	content string
	mode    os.FileMode
}

func fileRead(fs afero.Fs, path string) (string, os.FileInfo, error) {
	stat, err := fs.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("unable to stat file %s: %w", path, err)
	}
	b, err := afero.ReadFile(fs, path)
	if err != nil {
		return "", nil, fmt.Errorf("unable to read file %s: %w", path, err)
	}
	return string(b), stat, nil
}

func stringRC(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func p[T any](v T) *T {
	return &v
}
