package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/cloudboss/easyto/pkg/initial/collections"
)

type s3API interface {
	GetObject(context.Context, *s3.GetObjectInput,
		...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjects(context.Context, *s3.ListObjectsInput,
		...func(*s3.Options)) (*s3.ListObjectsOutput, error)
}

type S3Client interface {
	GetObjectList(bucket, keyPrefix string) (collections.WritableList, error)
	GetObjectMap(bucket, keyPrefix string) (map[string]string, error)
	GetObjectValue(bucket, keyPrefix string) ([]byte, error)
}

func NewS3Client(cfg aws.Config) S3Client {
	return &s3Client{
		api: s3.NewFromConfig(cfg),
	}
}

type s3Client struct {
	api s3API
}

func (s *s3Client) GetObjectList(bucket, keyPrefix string) (collections.WritableList, error) {
	objects, err := s.listObjects(bucket, keyPrefix)
	if err != nil {
		return nil, err
	}
	return s.toList(objects, bucket, keyPrefix), nil
}

func (s *s3Client) GetObjectMap(bucket, key string) (map[string]string, error) {
	object, err := s.getObject(bucket, key)
	if err != nil {
		return nil, err
	}
	defer object.Body.Close()
	m := make(map[string]string)
	err = json.NewDecoder(object.Body).Decode(&m)
	if err != nil {
		s3URL := "s3://" + bucket + "/" + key
		return nil, fmt.Errorf("unable to decode map from object at %s: %w", s3URL, err)
	}
	return m, nil
}

func (s *s3Client) GetObjectValue(bucket, key string) ([]byte, error) {
	object, err := s.getObject(bucket, key)
	if err != nil {
		return nil, err
	}
	defer object.Body.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, object.Body)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *s3Client) getObject(bucket, key string) (*s3.GetObjectOutput, error) {
	object, err := s.api.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: p(bucket),
		Key:    p(key),
	})
	if err != nil {
		s3URL := "s3://" + bucket + "/" + key
		return nil, fmt.Errorf("unable to get object at %s: %w", s3URL, err)
	}
	if object.Body == nil {
		return nil, fmt.Errorf("object %s has no body", key)
	}
	return object, nil
}

func (s *s3Client) listObjects(bucket, keyPrefix string) ([]types.Object, error) {
	var (
		objects []types.Object
		marker  *string
	)
	for {
		out, err := s.api.ListObjects(context.Background(), &s3.ListObjectsInput{
			Bucket: p(bucket),
			Prefix: p(keyPrefix),
			Marker: marker,
		})
		if err != nil {
			s3URL := "s3://" + bucket + "/" + keyPrefix
			return nil, fmt.Errorf("unable to list objects at %s: %w", s3URL, err)
		}
		objects = append(objects, out.Contents...)
		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		marker = objects[len(objects)-1].Key
	}
	return objects, nil
}

func (s *s3Client) toList(objects []types.Object, bucket, keyPrefix string) collections.WritableList {
	list := collections.WritableList{}
	for _, object := range objects {
		// Skip any objects that are "folders".
		if strings.HasSuffix(*object.Key, "/") {
			continue
		}
		if !strings.HasPrefix(*object.Key, keyPrefix) {
			continue
		}
		key := *object.Key
		if len(keyPrefix) > 0 {
			// If key and keyPrefix are the same, this will result in an empty
			// string, which enables the destination to become the filename
			// instead of directory when calling the Write method on the returned
			// List. This is a special case for retrieving a single object.
			fields := strings.Split(key, keyPrefix)
			key = fields[1]
		}
		s3Object := &S3Object{bucket: bucket, api: s.api, key: *object.Key}
		listEntry := &collections.WritableListEntry{Path: key, Value: s3Object}
		list = append(list, listEntry)
	}
	return list
}

type S3Object struct {
	api    s3API
	bucket string
	key    string
	object *s3.GetObjectOutput
}

func (s *S3Object) Read(p []byte) (n int, err error) {
	if s.object == nil {
		s.object, err = s.api.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: &s.bucket,
			Key:    &s.key,
		})
		if err != nil {
			return 0, fmt.Errorf("unable to get object %s: %w", s.key, err)
		}
	}
	if s.object.Body == nil {
		return 0, fmt.Errorf("object %s has no body", s.key)
	}
	return s.object.Body.Read(p)
}

func (s *S3Object) Close() error {
	if s.object == nil || s.object.Body == nil {
		return nil
	}
	return s.object.Body.Close()
}
