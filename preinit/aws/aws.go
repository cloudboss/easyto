package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type Connection interface {
	SSMClient() SSMClient
}

type connection struct {
	cfg       aws.Config
	ssmClient SSMClient
}

func NewConnection(region string) (*connection, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}
	return &connection{cfg: cfg}, nil
}

func (c *connection) SSMClient() SSMClient {
	if c.ssmClient == nil {
		c.ssmClient = NewSSMClient(c.cfg)
	}
	return c.ssmClient
}

func p[T any](v T) *T {
	return &v
}
