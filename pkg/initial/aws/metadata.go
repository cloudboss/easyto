package aws

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/cloudboss/easyto/pkg/initial/vmspec"
	yaml "github.com/goccy/go-yaml"
)

var (
	imdsClient = imds.New(imds.Options{})
)

func GetUserData() (*vmspec.VMSpec, error) {
	spec := &vmspec.VMSpec{}

	out, err := imdsClient.GetUserData(context.Background(), &imds.GetUserDataInput{})
	if err != nil {
		slog.Warn("Unable to get user data", "error", err)
		return spec, nil
	}

	err = yaml.NewDecoder(out.Content).Decode(spec)
	return spec, err
}

func GetSSHPubKey() (string, error) {
	resp, err := imdsClient.GetMetadata(context.Background(), &imds.GetMetadataInput{
		Path: "public-keys/0/openssh-key",
	})
	if err != nil {
		return "", fmt.Errorf("error getting SSH public key from metadata: %w", err)
	}

	content, err := io.ReadAll(resp.Content)
	if err != nil {
		return "", fmt.Errorf("error reading SSH public key: %w", err)
	}

	return string(content), nil
}

func GetRegion() (string, error) {
	resp, err := imdsClient.GetMetadata(context.Background(), &imds.GetMetadataInput{
		Path: "placement/region",
	})
	if err != nil {
		return "", fmt.Errorf("error getting region from metadata: %w", err)
	}

	content, err := io.ReadAll(resp.Content)
	if err != nil {
		return "", fmt.Errorf("error reading region: %w", err)
	}

	return string(content), nil
}
