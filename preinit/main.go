package main

import (
	"fmt"
	"os"

	"github.com/cloudboss/easyto/preinit/preinit"
)

func main() {
	err := preinit.DoIt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up init: %s\n", err)
		os.Exit(1)
	}
}
