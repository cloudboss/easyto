package copybuilder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cloudboss/easyto/pkg/constants"
)

var (
	ErrNotFound = errors.New("builder AMI not found")
)

type DescribeImagesAPI interface {
	DescribeImages(
		ctx context.Context,
		params *ec2.DescribeImagesInput,
		optFns ...func(*ec2.Options),
	) (*ec2.DescribeImagesOutput, error)
}

type CopyImageAPI interface {
	CopyImage(
		ctx context.Context,
		params *ec2.CopyImageInput,
		optFns ...func(*ec2.Options),
	) (*ec2.CopyImageOutput, error)
}

type ImageWaiter interface {
	Wait(
		ctx context.Context,
		params *ec2.DescribeImagesInput,
		maxWaitDur time.Duration,
		optFns ...func(*ec2.ImageAvailableWaiterOptions),
	) error
}

type Config struct {
	SourceRegion string
	DestRegion   string
	Version      string
	Name         string
	CopyTags     bool
	Wait         bool
	Output       io.Writer
}

type Result struct {
	SourceAMI  string
	SourceName string
	DestAMI    string
	DestName   string
}

func Copy(ctx context.Context, cfg Config) (*Result, error) {
	sourceCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.SourceRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for source region: %w", err)
	}
	sourceClient := ec2.NewFromConfig(sourceCfg)

	destRegion := cfg.DestRegion
	if destRegion == "" {
		destCfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load default AWS config: %w", err)
		}
		destRegion = destCfg.Region
	}

	destCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(destRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for dest region: %w", err)
	}
	destClient := ec2.NewFromConfig(destCfg)

	var waiter ImageWaiter
	if cfg.Wait {
		waiter = ec2.NewImageAvailableWaiter(destClient)
	}

	return CopyWithClients(ctx, cfg, sourceClient, destClient, waiter)
}

func CopyWithClients(
	ctx context.Context,
	cfg Config,
	sourceClient DescribeImagesAPI,
	destClient CopyImageAPI,
	waiter ImageWaiter,
) (*Result, error) {
	amiName := constants.AMIPatternCloudboss + cfg.Version
	log(cfg.Output, "Looking for AMI %s in %s...\n", amiName, cfg.SourceRegion)

	sourceAMI, err := findAMI(ctx, sourceClient, amiName)
	if err != nil {
		return nil, err
	}
	log(cfg.Output, "Found source AMI %s\n", *sourceAMI.ImageId)

	destName := cfg.Name
	if destName == "" {
		destName = *sourceAMI.Name
	}

	log(cfg.Output, "Copying AMI to %s as %s...\n", cfg.DestRegion, destName)

	copyOutput, err := destClient.CopyImage(ctx, &ec2.CopyImageInput{
		Name:          aws.String(destName),
		SourceImageId: sourceAMI.ImageId,
		SourceRegion:  aws.String(cfg.SourceRegion),
		CopyImageTags: aws.Bool(cfg.CopyTags),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to copy AMI: %w", err)
	}

	log(cfg.Output, "Created AMI %s\n", *copyOutput.ImageId)

	result := &Result{
		SourceAMI:  *sourceAMI.ImageId,
		SourceName: *sourceAMI.Name,
		DestAMI:    *copyOutput.ImageId,
		DestName:   destName,
	}

	if waiter != nil {
		log(cfg.Output, "Waiting for AMI to become available...\n")
		err = waiter.Wait(ctx, &ec2.DescribeImagesInput{
			ImageIds: []string{*copyOutput.ImageId},
		}, 30*time.Minute)
		if err != nil {
			return result, fmt.Errorf("AMI copy initiated but wait failed: %w", err)
		}
		log(cfg.Output, "AMI is now available\n")
	}

	return result, nil
}

func log(w io.Writer, format string, args ...any) {
	if w != nil {
		fmt.Fprintf(w, format, args...)
	}
}

func findAMI(ctx context.Context, client DescribeImagesAPI, name string) (*ec2types.Image, error) {
	output, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{name},
			},
		},
		Owners: []string{constants.AWSAccountCloudboss},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe images: %w", err)
	}

	if len(output.Images) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}

	return &output.Images[0], nil
}
