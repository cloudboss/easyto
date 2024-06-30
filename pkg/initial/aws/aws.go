package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type Connection interface {
	ASMClient() ASMClient
	SSMClient() SSMClient
	S3Client() S3Client
}

type connection struct {
	asmClient ASMClient
	cfg       aws.Config
	ssmClient SSMClient
	s3Client  S3Client
}

func NewConnection(region string) (*connection, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}
	return &connection{cfg: cfg}, nil
}

func (c *connection) ASMClient() ASMClient {
	if c.asmClient == nil {
		c.asmClient = NewASMClient(c.cfg)
	}
	return c.asmClient
}

func (c *connection) SSMClient() SSMClient {
	if c.ssmClient == nil {
		c.ssmClient = NewSSMClient(c.cfg)
	}
	return c.ssmClient
}

func (c *connection) S3Client() S3Client {
	if c.s3Client == nil {
		c.s3Client = NewS3Client(c.cfg)
	}
	return c.s3Client
}

func p[T any](v T) *T {
	return &v
}
