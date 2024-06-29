package tree

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "easyto",
		Short: "A container image conversion tool",
	}
)

func Execute() error {
	return rootCmd.Execute()
}
