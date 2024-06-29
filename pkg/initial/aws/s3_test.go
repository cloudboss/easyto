package aws

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/cloudboss/easyto/pkg/initial/maps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type stringReadCloser struct {
	strings.Reader
}

func NewStringReadCloser(s string) *stringReadCloser {
	return &stringReadCloser{*strings.NewReader(s)}
}

func (s *stringReadCloser) Close() error {
	return nil
}

func (s *stringReadCloser) Read(b []byte) (n int, err error) {
	return s.Read(b)
}

type mockS3Client struct {
	fs afero.Fs
}

func (m *mockS3Client) ListObjects(bucket, keyPrefix string) (*s3ObjectList, error) {
	return nil, nil
}

func (m *mockS3Client) CopyObjects(objects *s3ObjectList, dest, subPath string, uid, gid int) error {
	w := func(dest string, value types.Object, uid, gid int) (err error) {
		content := fmt.Sprintf("%s-value", filepath.Base(*value.Key))
		out := &s3.GetObjectOutput{
			Body: NewStringReadCloser(content),
		}
		defer func() {
			closeErr := out.Body.Close()
			if closeErr != nil && err == nil {
				err = closeErr
			}
		}()
		return writeReader(m.fs, dest, out.Body, uid, gid)
	}
	return maps.Write(objects.Map(), w, dest, subPath, uid, gid)
}

func Test_S3Client_CopyObjects(t *testing.T) {
	testCases := []struct {
		dest    string
		dirs    []string
		err     error
		objects []types.Object
		prefix  string
		result  map[string][]byte
		subPath string
	}{
		{
			dirs:    []string{},
			objects: []types.Object{},
			result:  map[string][]byte{},
		},
		{
			dirs: []string{},
			objects: []types.Object{
				{
					Key: p("a1"),
				},
			},
			result: map[string][]byte{
				"a1": []byte("a1-value"),
			},
		},
		{
			dest: "zzz",
			dirs: []string{"zzz", "zzz/b1"},
			objects: []types.Object{
				{
					Key: p("b1/c1"),
				},
			},
			result: map[string][]byte{
				"zzz/b1/c1": []byte("c1-value"),
			},
		},
		{
			dest:   "hhh",
			dirs:   []string{"hhh"},
			prefix: "e1/f1",
			objects: []types.Object{
				{
					Key: p("e1/f1/g1"),
				},
				{
					Key: p("e1/f1/g2"),
				},
				{
					Key: p("e1/h1"),
				},
				{
					Key: p("e1/h2"),
				},
			},
			result: map[string][]byte{
				"hhh/g1": []byte("g1-value"),
				"hhh/g2": []byte("g2-value"),
			},
		},
		{
			dest: "jjj",
			dirs: []string{
				"jjj",
				"jjj/k1",
				"jjj/k1/l1",
				"jjj/k1/l1/m1",
				"jjj/k1/l1/m2",
			},
			prefix: "",
			objects: []types.Object{
				{
					Key: p("k1/l1/m1/n1"),
				},
				{
					Key: p("k1/l1/m1/n2"),
				},
				{
					Key: p("k1/l1/m2/o1"),
				},
				{
					Key: p("k1/l1/m2/o2"),
				},
				{
					Key: p("x/y"),
				},
			},
			result: map[string][]byte{
				"jjj/k1/l1/m1/n1": []byte("n1-value"),
				"jjj/k1/l1/m1/n2": []byte("n2-value"),
				"jjj/k1/l1/m2/o1": []byte("o1-value"),
				"jjj/k1/l1/m2/o2": []byte("o2-value"),
			},
		},
		{
			dest:   "jjj",
			dirs:   []string{"jjj", "jjj/m1", "jjj/m2"},
			prefix: "k1/l1",
			objects: []types.Object{
				{
					Key: p("k1/l1/m1/n1"),
				},
				{
					Key: p("k1/l1/m1/n2"),
				},
				{
					Key: p("k1/l1/m2/o1"),
				},
				{
					Key: p("k1/l1/m2/o2"),
				},
				{
					Key: p("x/y"),
				},
			},
			result: map[string][]byte{
				"jjj/m1/n1": []byte("n1-value"),
				"jjj/m1/n2": []byte("n2-value"),
				"jjj/m2/o1": []byte("o1-value"),
				"jjj/m2/o2": []byte("o2-value"),
			},
		},
		{
			dest:   "jjj",
			dirs:   []string{"jjj", "jjj/m1", "jjj/m2"},
			prefix: "k1/l1",
			objects: []types.Object{
				{
					// Item is filtered out because
					// it has children n1 & n2.
					Key: p("k1/l1/m1"),
				},
				{
					Key: p("k1/l1/m1/n1"),
				},
				{
					Key: p("k1/l1/m1/n2"),
				},
				{
					Key: p("k1/l1/m2/o1"),
				},
				{
					Key: p("k1/l1/m2/o2"),
				},
				{
					Key: p("x/y"),
				},
			},
			result: map[string][]byte{
				"jjj/m1/n1": []byte("n1-value"),
				"jjj/m1/n2": []byte("n2-value"),
				"jjj/m2/o1": []byte("o1-value"),
				"jjj/m2/o2": []byte("o2-value"),
			},
		},
	}

	for _, tc := range testCases {
		fs := afero.NewMemMapFs()
		client := &mockS3Client{fs: fs}
		objects := NewS3Objects(tc.objects, "abc", tc.prefix)
		err := client.CopyObjects(objects, tc.dest, tc.subPath, -1, -1)
		assert.Equal(t, tc.err, err)

		for _, dir := range tc.dirs {
			t.Run(fmt.Sprintf("directory %s", dir), func(t *testing.T) {
				dirExists, err := afero.DirExists(fs, dir)
				assert.True(t, dirExists, "directory %s does not exist", dir)
				assert.Nil(t, err)
			})
		}

		if len(tc.dest) == 0 {
			continue
		}

		for pth, contents := range tc.result {
			t.Run(fmt.Sprintf("file %s", tc.dest), func(t *testing.T) {
				fileContainsBytes, err := afero.FileContainsBytes(fs, pth, contents)
				assert.True(t, fileContainsBytes,
					"file %s does not contain expected contents", pth)
				assert.Nil(t, err, "error was not nil: %s", err)
			})
		}
	}
}

func Test_objectsToMap(t *testing.T) {
	testCases := []struct {
		objects []types.Object
		prefix  string
		result  map[string]any
		err     error
	}{
		{
			objects: []types.Object{},
			prefix:  "",
			result:  map[string]any{},
			err:     nil,
		},
		{
			objects: []types.Object{
				{
					Key: p("x1"),
				},
			},
			prefix: "x1",
			result: map[string]any{},
			err:    nil,
		},
		{
			objects: []types.Object{
				{
					Key: p("z1"),
				},
			},
			prefix: "x1",
			result: map[string]any{},
			err:    nil,
		},
		{
			objects: []types.Object{
				{
					Key: p("a1"),
				},
			},
			prefix: "",
			result: map[string]any{
				"a1": types.Object{Key: p("a1")},
			},
			err: nil,
		},
		{
			objects: []types.Object{
				{
					Key: p("b1"),
				},
				{
					Key: p("c1/d1"),
				},
				{
					Key: p("c1/d2"),
				},
			},
			prefix: "",
			result: map[string]any{
				"b1": types.Object{Key: p("b1")},
				"c1": map[string]any{
					"d1": types.Object{Key: p("c1/d1")},
					"d2": types.Object{Key: p("c1/d2")},
				},
			},
			err: nil,
		},
		{
			objects: []types.Object{
				{
					Key: p("j1"),
				},
				{
					Key: p("k1/l1"),
				},
				{

					Key: p("k1/l2"),
				},
			},
			prefix: "k1",
			result: map[string]any{
				"l1": types.Object{Key: p("k1/l1")},
				"l2": types.Object{Key: p("k1/l2")},
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		actual := objectsToMap(tc.objects, tc.prefix)
		assert.EqualValues(t, tc.result, actual)
	}
}

func Test_writeReader(t *testing.T) {
	testCases := []struct {
		contents string
		dest     string
		dirs     []string
	}{
		{

			contents: "a1-value",
			dest:     "a1",
			dirs:     []string{},
		},
		{

			contents: "d1-value",
			dest:     "b1/c1/d1",
			dirs:     []string{"b1", "b1/c1"},
		},
	}
	for _, tc := range testCases {
		fs := afero.NewMemMapFs()

		err := writeReader(fs, tc.dest, strings.NewReader(tc.contents), 0, 0)
		assert.Nil(t, err, "error was not nil: %s", err)

		for _, dir := range tc.dirs {
			t.Run(dir, func(t *testing.T) {
				dirExists, err := afero.DirExists(fs, dir)
				assert.True(t, dirExists, "directory %s does not exist", dir)
				assert.Nil(t, err)
			})
		}

		if len(tc.dest) == 0 {
			continue
		}

		t.Run(tc.dest, func(t *testing.T) {
			fileContainsBytes, err := afero.FileContainsBytes(fs, tc.dest, []byte(tc.contents))
			assert.True(t, fileContainsBytes,
				"file %s does not contain expected contents", tc.dest)
			assert.Nil(t, err, "error was not nil: %s", err)
		})
	}
}
