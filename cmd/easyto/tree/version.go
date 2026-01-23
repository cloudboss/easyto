package tree

import (
	"github.com/spf13/cobra"

	"github.com/cloudboss/easyto/pkg/constants"
)

var (
	VersionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show the version of easyto",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(constants.ETVersion)
		},
	}
)
