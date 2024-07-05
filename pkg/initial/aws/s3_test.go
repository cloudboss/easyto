package aws

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var errObjectNotFound = errors.New("object not found")

type mockS3API struct {
	bucketObjects map[string]string
}

func (s *mockS3API) GetObject(ctx context.Context, in *s3.GetObjectInput,
	opt ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	content, ok := s.bucketObjects[*in.Key]
	if !ok {
		return nil, errObjectNotFound
	}
	out := &s3.GetObjectOutput{
		Body: stringRC(content),
	}
	return out, nil
}

func (s *mockS3API) ListObjects(ctx context.Context, in *s3.ListObjectsInput,
	opt ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	objects := []types.Object{}
	for k := range s.bucketObjects {
		objects = append(objects, types.Object{
			Key: p(k),
		})
	}
	out := &s3.ListObjectsOutput{
		Contents: objects,
	}
	return out, nil
}

func Test_S3Client_GetObjectList(t *testing.T) {
	testCases := []struct {
		description   string
		bucketObjects map[string]string
		dest          string
		keyPrefix     string
		result        []file
		err           error
	}{
		{
			description: "Single object",
			bucketObjects: map[string]string{
				"b1/c1": "c1-value",
			},
			dest:      "/abc",
			keyPrefix: "b1",
			result: []file{
				{
					name:    "/abc/c1",
					content: "c1-value",
					mode:    0644,
				},
			},
		},
		{
			description: "Nested objects",
			bucketObjects: map[string]string{
				"b1/c1":    "c1-value",
				"b1/d1/e1": "e1-value",
			},
			dest:      "/abc",
			keyPrefix: "b1",
			result: []file{
				{
					name:    "/abc/c1",
					content: "c1-value",
					mode:    0644,
				},
				{
					name: "/abc/d1",
					mode: 0755 | os.ModeDir,
				},
				{
					name:    "/abc/d1/e1",
					content: "e1-value",
					mode:    0644,
				},
			},
		},
		{
			description: "Same key and prefix",
			bucketObjects: map[string]string{
				"b1/d1/e1": "e1-value",
			},
			dest:      "/abc",
			keyPrefix: "b1/d1/e1",
			result: []file{
				{
					name:    "/abc",
					content: "e1-value",
					mode:    0644,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			client := s3Client{
				api: &mockS3API{tc.bucketObjects},
			}
			m, err := client.GetObjectList("thebucket", tc.keyPrefix)
			assert.NoError(t, err)
			err = m.Write(fs, tc.dest, 0, 0, false)
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

func Test_S3Client_GetObjectMap(t *testing.T) {
	testCases := []struct {
		description   string
		bucketObjects map[string]string
		key           string
		result        map[string]string
		err           bool
	}{
		{
			description: "Object is not map",
			bucketObjects: map[string]string{
				"a/b/c": "not-json",
			},
			key: "a/b/c",
			err: true,
		},
		{
			description: "Object is nested map",
			bucketObjects: map[string]string{
				"a/b": `{"abc": "123", "def": {"ghi": "789"}}`,
			},
			key: "a/b",
			err: true,
		},
		{
			description: "Object is valid map",
			bucketObjects: map[string]string{
				"a/b": `{"abc": "123", "def": "456"}`,
			},
			key: "a/b",
			result: map[string]string{
				"abc": "123",
				"def": "456",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			client := s3Client{
				api: &mockS3API{tc.bucketObjects},
			}
			m, err := client.GetObjectMap("thebucket", tc.key)
			if tc.err {
				assert.Error(t, err)
			}
			assert.Equal(t, tc.result, m)
		})
	}
}

func Test_S3Client_GetObjectValue(t *testing.T) {
	testCases := []struct {
		description   string
		bucketObjects map[string]string
		key           string
		result        []byte
		err           error
	}{
		{
			description: "Object not found",
			bucketObjects: map[string]string{
				"a/b": "Thirty for remove plenty regard you summer though",
			},
			key: "x/y/z",
			err: errObjectNotFound,
		},
		{
			description: "Single object",
			bucketObjects: map[string]string{
				"a/b": "Thirty for remove plenty regard you summer though",
			},
			key:    "a/b",
			result: []byte("Thirty for remove plenty regard you summer though"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			client := s3Client{
				api: &mockS3API{tc.bucketObjects},
			}
			b, err := client.GetObjectValue("thebucket", tc.key)
			assert.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.result, b)
		})
	}
}
