package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/cloudboss/easyto/pkg/initial/initial"
)

func main() {
	err := initial.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up init: %s\n", err)
	}

	// Give console output time to catch up
	// so we can see if there was an error.
	time.Sleep(5 * time.Second)

	// Time to power down no matter what.
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
