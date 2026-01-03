package tree

import (
	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:   "easyto",
		Short: "A container image conversion tool",
	}
)

func init() {
	RootCmd.AddCommand(AMICmd)
}
