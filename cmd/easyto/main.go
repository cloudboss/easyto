package main

import (
	"fmt"
	"os"

	"github.com/cloudboss/easyto/cmd/easyto/tree"
)

func main() {
	if err := tree.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
