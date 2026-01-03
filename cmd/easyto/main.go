package main

import (
	"os"

	"github.com/cloudboss/easyto/cmd/easyto/tree"
)

func main() {
	if err := tree.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
