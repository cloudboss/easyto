package tree

import (
	"context"
	"os"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/cloudboss/easyto/pkg/copybuilder"
	"github.com/spf13/cobra"
)

var (
	copyBuilderCfg = &copyBuilderConfig{}
	CopyBuilderCmd = &cobra.Command{
		Use:   "copy-builder",
		Short: "Copy the official easyto builder AMI to your account/region",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			ctx := context.Background()

			cfg := copybuilder.Config{
				SourceRegion: copyBuilderCfg.sourceRegion,
				DestRegion:   copyBuilderCfg.destRegion,
				Version:      copyBuilderCfg.version,
				Name:         copyBuilderCfg.name,
				CopyTags:     copyBuilderCfg.copyTags,
				Wait:         copyBuilderCfg.wait,
				Public:       copyBuilderCfg.public,
				Output:       os.Stdout,
			}

			result, err := copybuilder.Copy(ctx, cfg)
			if err != nil {
				return err
			}

			cmd.Printf("Copied %s (%s) to %s (%s)\n",
				result.SourceName, result.SourceAMI,
				result.DestName, result.DestAMI)
			return nil
		},
	}
)

type copyBuilderConfig struct {
	sourceRegion string
	destRegion   string
	version      string
	name         string
	copyTags     bool
	wait         bool
	public       bool
}

func init() {
	CopyBuilderCmd.Flags().StringVar(&copyBuilderCfg.sourceRegion, "source-region", "us-east-1",
		"AWS region where the official builder AMI is located.")

	CopyBuilderCmd.Flags().StringVar(&copyBuilderCfg.destRegion, "dest-region", "",
		"AWS region to copy the AMI to. Defaults to your configured AWS region.")

	CopyBuilderCmd.Flags().StringVar(&copyBuilderCfg.version, "version", constants.ETVersion,
		"Version of the builder AMI to copy.")

	CopyBuilderCmd.Flags().StringVar(&copyBuilderCfg.name, "name", "",
		"Name for the copied AMI. Defaults to the source AMI name.")

	CopyBuilderCmd.Flags().BoolVar(&copyBuilderCfg.copyTags, "copy-tags", true,
		"Copy tags from the source AMI.")

	CopyBuilderCmd.Flags().BoolVar(&copyBuilderCfg.wait, "wait", true,
		"Wait for the AMI copy to complete.")

	CopyBuilderCmd.Flags().BoolVar(&copyBuilderCfg.public, "public", false,
		"Make the copied AMI and its snapshots public.")
}
