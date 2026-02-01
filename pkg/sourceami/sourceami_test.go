package sourceami

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEC2Client struct {
	images []ec2types.Image
	err    error
}

func (m *mockEC2Client) DescribeImages(
	ctx context.Context,
	input *ec2.DescribeImagesInput,
	opts ...func(*ec2.Options),
) (*ec2.DescribeImagesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ec2.DescribeImagesOutput{Images: m.images}, nil
}

func TestLatestImage(t *testing.T) {
	testCases := []struct {
		name        string
		images      []ec2types.Image
		expected    string
		expectedErr error
	}{
		{
			name:        "Empty list",
			images:      []ec2types.Image{},
			expectedErr: errNotFound,
		},
		{
			name: "Single image",
			images: []ec2types.Image{
				{ImageId: aws.String("ami-123"), CreationDate: aws.String("2024-01-01T00:00:00.000Z")},
			},
			expected: "ami-123",
		},
		{
			name: "Multiple images returns latest",
			images: []ec2types.Image{
				{ImageId: aws.String("ami-old"), CreationDate: aws.String("2024-01-01T00:00:00.000Z")},
				{ImageId: aws.String("ami-newest"), CreationDate: aws.String("2024-03-01T00:00:00.000Z")},
				{ImageId: aws.String("ami-middle"), CreationDate: aws.String("2024-02-01T00:00:00.000Z")},
			},
			expected: "ami-newest",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := latestImage(tc.images)
			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestImageCmp(t *testing.T) {
	older := ec2types.Image{CreationDate: aws.String("2024-01-01T00:00:00.000Z")}
	newer := ec2types.Image{CreationDate: aws.String("2024-02-01T00:00:00.000Z")}

	assert.Equal(t, -1, imageCmp(older, newer))
	assert.Equal(t, 1, imageCmp(newer, older))
	assert.Equal(t, 0, imageCmp(older, older))
}

func TestGetSourceAMI(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		client      *mockEC2Client
		request     sourceAMIRequest
		expected    string
		expectedErr error
	}{
		{
			name: "Found image",
			client: &mockEC2Client{
				images: []ec2types.Image{
					{ImageId: aws.String("ami-found"), CreationDate: aws.String("2024-01-01T00:00:00.000Z")},
				},
			},
			request:  sourceAMIRequest{name: "test-ami", searchAWSAccounts: []string{"123456789"}},
			expected: "ami-found",
		},
		{
			name:        "No images found",
			client:      &mockEC2Client{images: []ec2types.Image{}},
			request:     sourceAMIRequest{name: "nonexistent"},
			expectedErr: errNotFound,
		},
		{
			name:        "API error",
			client:      &mockEC2Client{err: errors.New("API error")},
			request:     sourceAMIRequest{name: "test-ami"},
			expectedErr: errors.New("API error"),
		},
		{
			name: "Default architecture is x86_64",
			client: &mockEC2Client{
				images: []ec2types.Image{
					{ImageId: aws.String("ami-x86"), CreationDate: aws.String("2024-01-01T00:00:00.000Z")},
				},
			},
			request:  sourceAMIRequest{name: "test-ami"},
			expected: "ami-x86",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getSourceAMI(ctx, tc.client, tc.request)
			if tc.expectedErr != nil {
				require.Error(t, err)
				if errors.Is(tc.expectedErr, errNotFound) {
					assert.ErrorIs(t, err, errNotFound)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetSourceAMISlow(t *testing.T) {
	ctx := context.Background()

	client := &mockEC2Client{
		images: []ec2types.Image{
			{ImageId: aws.String("ami-debian"), CreationDate: aws.String("2024-01-01T00:00:00.000Z")},
		},
	}

	result, err := getSourceAMISlow(ctx, client, "")
	require.NoError(t, err)
	assert.Equal(t, "ami-debian", result)
}

func TestResolveOverride(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		client       *mockEC2Client
		builderImage string
		expectedAMI  string
		expectedMode int
		expectedErr  bool
	}{
		{
			name:         "Direct AMI ID",
			client:       &mockEC2Client{},
			builderImage: "ami-direct123",
			expectedAMI:  "ami-direct123",
			expectedMode: ModeSlow,
		},
		{
			name: "Name pattern lookup",
			client: &mockEC2Client{
				images: []ec2types.Image{
					{ImageId: aws.String("ami-looked-up"), CreationDate: aws.String("2024-01-01T00:00:00.000Z")},
				},
			},
			builderImage: "my-custom-ami-*",
			expectedAMI:  "ami-looked-up",
			expectedMode: ModeSlow,
		},
		{
			name:         "Name pattern not found",
			client:       &mockEC2Client{images: []ec2types.Image{}},
			builderImage: "nonexistent-pattern",
			expectedErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := resolveOverride(ctx, tc.client, tc.builderImage)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedAMI, result.AMI)
			assert.Equal(t, tc.expectedMode, result.Mode)
		})
	}
}
