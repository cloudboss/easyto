package sourceami

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cloudboss/easyto/pkg/constants"
)

const (
	ModeFast = iota
	ModeSlow

	archX86_64 = "x86_64"
)

var errNotFound = errors.New("source image not found")

type sourceAMIRequest struct {
	name              string
	architecture      string
	searchAWSAccounts []string
}

type Response struct {
	Mode int
	AMI  string
}

func getSourceAMI(
	ctx context.Context,
	client ec2.DescribeImagesAPIClient,
	request sourceAMIRequest,
) (string, error) {
	arch := request.architecture
	if arch == "" {
		arch = archX86_64
	}
	output, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []ec2types.Filter{
			{
				Name:   p("name"),
				Values: []string{request.name},
			},
			{
				Name:   p("architecture"),
				Values: []string{arch},
			},
		},
		Owners: request.searchAWSAccounts,
	})
	if err != nil {
		return "", err
	}
	return latestImage(output.Images)
}

func latestImage(images []ec2types.Image) (string, error) {
	numImages := len(images)
	if numImages == 0 {
		return "", errNotFound
	}
	slices.SortFunc(images, imageCmp)
	return *images[numImages-1].ImageId, nil
}

func imageCmp(a, b ec2types.Image) int {
	creationDateA, _ := time.Parse(time.RFC3339, *a.CreationDate)
	creationDateB, _ := time.Parse(time.RFC3339, *b.CreationDate)
	return creationDateA.Compare(creationDateB)
}

func getSourceAMISlow(ctx context.Context, client ec2.DescribeImagesAPIClient, arch string) (string, error) {
	request := sourceAMIRequest{
		name:              constants.AMIPatternDebian,
		architecture:      arch,
		searchAWSAccounts: []string{constants.AWSAccountDebian},
	}
	return getSourceAMI(ctx, client, request)
}

// Resolve determines the source AMI and build mode.
// If builderImage is provided:
//   - If it looks like an AMI ID (ami-*), use it directly with slow mode
//   - Otherwise treat it as a name pattern and look it up
//
// If builderImage is empty, try the fast path (easyto builder AMI matching
// the given version) and fall back to slow path (Debian) if not found.
func Resolve(ctx context.Context, builderImage, version string) (*Response, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := ec2.NewFromConfig(cfg)

	if builderImage != "" {
		return resolveOverride(ctx, client, builderImage)
	}

	amiName := constants.AMIPatternCloudboss + version

	// Check user's own account and Cloudboss account for the easyto builder AMI
	ami, err := getSourceAMI(ctx, client, sourceAMIRequest{
		name:              amiName,
		searchAWSAccounts: []string{"self", constants.AWSAccountCloudboss},
	})
	if err == nil {
		return &Response{Mode: ModeFast, AMI: ami}, nil
	}
	if !errors.Is(err, errNotFound) {
		return nil, err
	}

	// Fall back to Debian (slow path)
	ami, err = getSourceAMISlow(ctx, client, "")
	if err != nil {
		return nil, err
	}
	return &Response{Mode: ModeSlow, AMI: ami}, nil
}

func resolveOverride(ctx context.Context, client ec2.DescribeImagesAPIClient, builderImage string) (*Response, error) {
	if strings.HasPrefix(builderImage, "ami-") {
		return &Response{
			Mode: ModeSlow,
			AMI:  builderImage,
		}, nil
	}
	ami, err := getSourceAMI(ctx, client, sourceAMIRequest{
		name:              builderImage,
		searchAWSAccounts: []string{},
	})
	if err != nil {
		return nil, err
	}
	return &Response{
		Mode: ModeSlow,
		AMI:  ami,
	}, nil
}

func p[T any](v T) *T {
	return &v
}
