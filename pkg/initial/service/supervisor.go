package service

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/spf13/afero"
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
	dirs, err := afero.ReadDir(fs, constants.DirETServices)
	if err != nil {
		return fmt.Errorf("unable to read directory %s: %w", constants.DirETServices, err)
	}

	for _, dir := range dirs {
		svc := dir.Name()
		switch svc {
		case "chrony":
			s.Services = append(s.Services, NewChronyService())
		case "ssh":
			s.Services = append(s.Services, NewSSHDService())
		default:
			slog.Warn("Unknown service", "service", svc)
		}
	}

	for _, service := range s.Services {
		err := service.Start()
		if err != nil {
			if service.Optional() {
				slog.Warn("Optional service failed to start", "service", service, "error", err)
				continue
			}
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

	didShutdownAll := false
	shutdownAll := func() {
		if didShutdownAll {
			return
		} else {
			didShutdownAll = true
		}

		slog.Info("Shutting down all processes")

		// Set the timer in case services do not exit.
		timeout.Reset(s.Timeout)

		// Send a SIGTERM to all services.
		s.Stop()
	}

	go func() {
		err := s.Main.Wait()
		if !(err == nil || errors.Is(err, syscall.ECHILD)) {
			slog.Error("Main process exited", "error", err)
		} else {
			slog.Info("Main process exited")
		}
		shutdownAll()
	}()

	go func() {
		for {
			pid, err := syscall.Wait4(-1, nil, 0, nil)
			slog.Debug("Reaped process", "pid", pid, "error", err)
			if err != nil && errors.Is(err, syscall.ECHILD) {
				// All processes have exited.
				break
			}
		}
		doneC <- struct{}{}
	}()

	stopped := false

	for !stopped {
		select {
		case <-poweroffC:
			slog.Info("Got poweroff signal")
			go shutdownAll()
		case <-doneC:
			slog.Info("All processes have exited")
			stopped = true
		case <-timeout.C:
			slog.Warn("Timeout waiting for graceful shutdown")
			s.Kill()
			stopped = true
		}
	}
}
