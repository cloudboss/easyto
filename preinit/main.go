package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/cloudboss/cb/cbinit/cbinit"
	// _ "github.com/cloudboss/punk/punk"
)

func main() {
	initSpec, err := cbinit.Setup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up init: %s\n", err)
		os.Exit(1)
	}
	err = cbinit.SwitchRoot("/newroot", initSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to run init: %s\n", err)
		runShell()
		os.Exit(1)
	}
}

func runShell() {
	cmd := exec.Command("/bin/sh")
	err := cmd.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running shell: %s\n", err)
	}
}
