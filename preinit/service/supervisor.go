package service

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// Signal sent by the "ACPI tiny power button" kernel driver.
	// It is assumed the kernel will be compiled to use it.
	SIGPWRBTN = syscall.Signal(0x26)
)

type Supervisor struct {
	Main     Service
	Services []Service
	Timeout  time.Duration
}

func (s *Supervisor) Start() error {
	for _, service := range s.Services {
		err := service.Start()
		if !(err == nil || service.Optional()) {
			return err
		}
	}
	return s.Main.Start()
}

func (s *Supervisor) Stop() {
	for _, service := range s.Services {
		service.Stop()
	}
	s.Main.Stop()
}

func (s *Supervisor) Kill() {
	for _, service := range s.Services {
		service.Kill()
	}
	s.Main.Kill()
}

func (s *Supervisor) Wait() {
	poweroffC := make(chan os.Signal, 1)
	signal.Notify(poweroffC, SIGPWRBTN)

	doneC := make(chan struct{}, 1)

	// Create a timeout with an unreachable duration
	// to be adjusted when it's time to shut down.
	forever := time.Duration(1<<63 - 1)
	timeout := time.NewTimer(forever)

	shutdownAll := func() {
		fmt.Println("Shutting down all processes")

		// Set the timer in case services do not exit.
		timeout.Reset(s.Timeout)

		// Send a SIGTERM to all services.
		s.Stop()

		for _, service := range s.Services {
			err := service.Wait()
			if err != nil {
				fmt.Printf("Process %s exited with error: %s\n", service, err)
			}
		}

		doneC <- struct{}{}
	}

	go func() {
		err := s.Main.Wait()
		if err != nil {
			fmt.Printf("Main process exited with error: %s\n", err)
		}
		shutdownAll()
	}()

	stopped := false

	for !stopped {
		select {
		case <-poweroffC:
			fmt.Println("Got poweroff signal")
			shutdownAll()
		case <-doneC:
			fmt.Println("All processes have exited")
			stopped = true
		case <-timeout.C:
			fmt.Println("Timeout waiting for graceful shutdown")
			s.Kill()
			stopped = true
		}
	}
}
