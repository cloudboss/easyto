package copybuilder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDescribeImagesClient struct {
	images []ec2types.Image
	err    error
}

func (m *mockDescribeImagesClient) DescribeImages(
	ctx context.Context,
	input *ec2.DescribeImagesInput,
	opts ...func(*ec2.Options),
) (*ec2.DescribeImagesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ec2.DescribeImagesOutput{Images: m.images}, nil
}

type mockCopyImageClient struct {
	imageID string
	err     error
}

func (m *mockCopyImageClient) CopyImage(
	ctx context.Context,
	input *ec2.CopyImageInput,
	opts ...func(*ec2.Options),
) (*ec2.CopyImageOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ec2.CopyImageOutput{ImageId: aws.String(m.imageID)}, nil
}

type mockWaiter struct {
	err error
}

func (m *mockWaiter) Wait(
	ctx context.Context,
	params *ec2.DescribeImagesInput,
	maxWaitDur time.Duration,
	optFns ...func(*ec2.ImageAvailableWaiterOptions),
) error {
	return m.err
}

func TestFindAMI(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		client      *mockDescribeImagesClient
		amiName     string
		expectedID  string
		expectedErr error
	}{
		{
			name: "Found image",
			client: &mockDescribeImagesClient{
				images: []ec2types.Image{
					{
						ImageId: aws.String("ami-123"),
						Name:    aws.String("test-ami"),
					},
				},
			},
			amiName:    "test-ami",
			expectedID: "ami-123",
		},
		{
			name:        "No images found",
			client:      &mockDescribeImagesClient{images: []ec2types.Image{}},
			amiName:     "nonexistent",
			expectedErr: ErrNotFound,
		},
		{
			name:        "API error",
			client:      &mockDescribeImagesClient{err: errors.New("API error")},
			amiName:     "test-ami",
			expectedErr: errors.New("failed to describe images"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := findAMI(ctx, tc.client, tc.amiName)
			if tc.expectedErr != nil {
				require.Error(t, err)
				if errors.Is(tc.expectedErr, ErrNotFound) {
					assert.ErrorIs(t, err, ErrNotFound)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedID, *result.ImageId)
		})
	}
}

func TestCopyWithClients(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		cfg            Config
		sourceClient   *mockDescribeImagesClient
		destClient     *mockCopyImageClient
		waiter         *mockWaiter
		expectedResult *Result
		expectedErr    string
	}{
		{
			name: "Successful copy without wait",
			cfg: Config{
				SourceRegion: "us-east-1",
				Version:      "v1.0.0",
			},
			sourceClient: &mockDescribeImagesClient{
				images: []ec2types.Image{
					{
						ImageId: aws.String("ami-source"),
						Name:    aws.String("ghcr.io--cloudboss--easyto-builder--v1.0.0"),
					},
				},
			},
			destClient: &mockCopyImageClient{imageID: "ami-dest"},
			waiter:     nil,
			expectedResult: &Result{
				SourceAMI:  "ami-source",
				SourceName: "ghcr.io--cloudboss--easyto-builder--v1.0.0",
				DestAMI:    "ami-dest",
				DestName:   "ghcr.io--cloudboss--easyto-builder--v1.0.0",
			},
		},
		{
			name: "Successful copy with custom name",
			cfg: Config{
				SourceRegion: "us-east-1",
				Version:      "v1.0.0",
				Name:         "my-custom-name",
			},
			sourceClient: &mockDescribeImagesClient{
				images: []ec2types.Image{
					{
						ImageId: aws.String("ami-source"),
						Name:    aws.String("ghcr.io--cloudboss--easyto-builder--v1.0.0"),
					},
				},
			},
			destClient: &mockCopyImageClient{imageID: "ami-dest"},
			waiter:     nil,
			expectedResult: &Result{
				SourceAMI:  "ami-source",
				SourceName: "ghcr.io--cloudboss--easyto-builder--v1.0.0",
				DestAMI:    "ami-dest",
				DestName:   "my-custom-name",
			},
		},
		{
			name: "Successful copy with wait",
			cfg: Config{
				SourceRegion: "us-east-1",
				Version:      "v1.0.0",
				Wait:         true,
			},
			sourceClient: &mockDescribeImagesClient{
				images: []ec2types.Image{
					{
						ImageId: aws.String("ami-source"),
						Name:    aws.String("ghcr.io--cloudboss--easyto-builder--v1.0.0"),
					},
				},
			},
			destClient: &mockCopyImageClient{imageID: "ami-dest"},
			waiter:     &mockWaiter{},
			expectedResult: &Result{
				SourceAMI:  "ami-source",
				SourceName: "ghcr.io--cloudboss--easyto-builder--v1.0.0",
				DestAMI:    "ami-dest",
				DestName:   "ghcr.io--cloudboss--easyto-builder--v1.0.0",
			},
		},
		{
			name: "Source AMI not found",
			cfg: Config{
				SourceRegion: "us-east-1",
				Version:      "v1.0.0",
			},
			sourceClient: &mockDescribeImagesClient{images: []ec2types.Image{}},
			destClient:   &mockCopyImageClient{imageID: "ami-dest"},
			expectedErr:  "builder AMI not found",
		},
		{
			name: "Copy fails",
			cfg: Config{
				SourceRegion: "us-east-1",
				Version:      "v1.0.0",
			},
			sourceClient: &mockDescribeImagesClient{
				images: []ec2types.Image{
					{
						ImageId: aws.String("ami-source"),
						Name:    aws.String("ghcr.io--cloudboss--easyto-builder--v1.0.0"),
					},
				},
			},
			destClient:  &mockCopyImageClient{err: errors.New("copy failed")},
			expectedErr: "failed to copy AMI",
		},
		{
			name: "Wait fails",
			cfg: Config{
				SourceRegion: "us-east-1",
				Version:      "v1.0.0",
				Wait:         true,
			},
			sourceClient: &mockDescribeImagesClient{
				images: []ec2types.Image{
					{
						ImageId: aws.String("ami-source"),
						Name:    aws.String("ghcr.io--cloudboss--easyto-builder--v1.0.0"),
					},
				},
			},
			destClient: &mockCopyImageClient{imageID: "ami-dest"},
			waiter:     &mockWaiter{err: errors.New("timeout")},
			expectedResult: &Result{
				SourceAMI:  "ami-source",
				SourceName: "ghcr.io--cloudboss--easyto-builder--v1.0.0",
				DestAMI:    "ami-dest",
				DestName:   "ghcr.io--cloudboss--easyto-builder--v1.0.0",
			},
			expectedErr: "AMI copy initiated but wait failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var waiter ImageWaiter
			if tc.waiter != nil {
				waiter = tc.waiter
			}

			result, err := CopyWithClients(ctx, tc.cfg, tc.sourceClient, tc.destClient, waiter)

			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
				if tc.expectedResult != nil {
					assert.Equal(t, tc.expectedResult, result)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
