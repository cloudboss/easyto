package aws

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/cloudboss/easyto/preinit/files"
	"github.com/cloudboss/easyto/preinit/maps"
	"github.com/spf13/afero"
)

type S3Client interface {
	ListObjects(bucket, keyPrefix string) (*s3ObjectList, error)
	CopyObjects(objects *s3ObjectList, dest, subPath string, uid, gid int) error
}

type s3Client struct {
	client *s3.Client
	fs     afero.Fs
}

type s3ObjectList struct {
	bucket string
	m      map[string]any
}

func NewS3Objects(objects []types.Object, bucket, keyPrefix string) *s3ObjectList {
	return &s3ObjectList{
		bucket: bucket,
		m:      objectsToMap(objects, keyPrefix),
	}
}

func (s *s3ObjectList) Map() map[string]any {
	return s.m
}

type S3Object struct {
	object       types.Object
	objectOutput *s3.GetObjectOutput
}

func NewS3Client(cfg aws.Config) S3Client {
	return &s3Client{
		client: s3.NewFromConfig(cfg),
		fs:     afero.NewOsFs(),
	}
}

func (s *s3Client) SetFS(fs afero.Fs) {
	s.fs = fs
}

func (s *s3Client) ListObjects(bucket, keyPrefix string) (*s3ObjectList, error) {
	list, err := s.client.ListObjects(context.Background(), &s3.ListObjectsInput{
		Bucket: p(bucket),
		Prefix: p(keyPrefix),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list objects at s3://%s/%s: %w", bucket, keyPrefix, err)
	}

	objects := NewS3Objects(list.Contents, bucket, keyPrefix)

	return objects, nil
}

func (s *s3Client) CopyObjects(objects *s3ObjectList, dest, subPath string, uid, gid int) error {
	w := func(fileDest string, value types.Object, uid, gid int) (err error) {
		out, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: &objects.bucket,
			Key:    value.Key,
		})
		if err != nil {
			return fmt.Errorf("unable to get object %s: %w", *value.Key, err)
		}
		defer func() {
			closeErr := out.Body.Close()
			if closeErr != nil && err == nil {
				err = closeErr
			}
		}()
		return writeReader(s.fs, fileDest, out.Body, uid, gid)
	}
	return maps.Write(objects.m, w, dest, subPath, uid, gid)
}

func objectsToMap(objects []types.Object, prefix string) map[string]any {
	if len(prefix) > 0 && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	oMap := map[string]any{}

	for _, obj := range objects {
		// Skip any objects that are "folders".
		if strings.HasSuffix(*obj.Key, "/") {
			continue
		}
		if !strings.HasPrefix(*obj.Key, prefix) {
			continue
		}
		key := *obj.Key
		if len(prefix) > 0 {
			fields := strings.Split(*obj.Key, prefix)
			key = fields[1]
		}
		if strings.Contains(key, "/") {
			newFields := strings.Split(key, "/")
			newPrefix := filepath.Join(prefix, newFields[0])
			oMap[newFields[0]] = objectsToMap(objects, newPrefix)
		} else {
			oMap[key] = obj
		}
	}

	return oMap
}

func writeReader(fs afero.Fs, dest string, value io.Reader, uid, gid int) (err error) {
	const (
		modeDir  = 0755
		modeFile = 0644
	)

	destDir := dest
	if !strings.HasSuffix(dest, "/") {
		destDir = filepath.Dir(dest)
	}

	err = files.Mkdirs(fs, destDir, uid, gid, modeDir)
	if err != nil {
		return err
	}

	if strings.HasSuffix(dest, "/") {
		return nil
	}

	f, err := fs.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, modeFile)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %w", dest, err)
	}
	defer func() {
		closeErr := f.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(f, value)

	if err != nil {
		return fmt.Errorf("unable to copy to %s: %w", dest, err)
	}

	err = fs.Chown(dest, uid, gid)
	if err != nil {
		return fmt.Errorf("unable to set permissions on file %s: %w",
			dest, err)
	}

	return nil
}
