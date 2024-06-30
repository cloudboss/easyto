package service

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

const (
	// Signal sent by the "ACPI tiny power button" kernel driver.
	// It is assumed the kernel will be compiled to use it.
	SIGPWRBTN = syscall.Signal(0x26)

	// Flag indicating process is a kernel thread, from include/linux/sched.h.
	PF_KTHREAD = 0x00200000
)

type Supervisor struct {
	Main           Service
	ReadonlyRootFS bool
	Services       []Service
	Timeout        time.Duration
}

func (s *Supervisor) Start() error {
	entries, err := afero.ReadDir(fs, constants.DirETServices)
	if !(err == nil || errors.Is(err, os.ErrNotExist)) {
		return fmt.Errorf("unable to read directory %s: %w", constants.DirETServices, err)
	}

	for _, entry := range entries {
		svc := entry.Name()
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

	// This needs to be done after services are started so that e.g. ssh-keygen can run.
	if s.ReadonlyRootFS {
		err = unix.Mount("", constants.DirRoot, "", syscall.MS_REMOUNT|syscall.MS_RDONLY, "")
		if err != nil {
			return fmt.Errorf("unable to remount root filesystem read-only: %w", err)
		}
	}

	return s.Main.Start()
}

func (s *Supervisor) Stop() {
	s.signal(syscall.SIGTERM)
}

func (s *Supervisor) Kill() {
	s.signal(syscall.SIGKILL)
}

func (s *Supervisor) signal(signal syscall.Signal) {
	// Ensure services know it is shutdown time so they don't restart.
	for _, service := range s.Services {
		service.Stop()
	}
	pids := s.pids()
	for _, pid := range pids {
		if pid == 1 {
			continue
		}
		unix.Kill(pid, signal)
	}
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

		// Set the timer in case processes do not exit.
		timeout.Reset(s.Timeout)

		// Send a SIGTERM to all running processes.
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
		// Don't start reaping processes until the main process has started,
		// otherwise the system may shut down before it starts, especially
		// in cases where there are no services besides the main process.
		s.Main.WaitStart()
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

// pids returns all current userspace PIDs. If there is an error reading /proc, the
// PIDs of the known services are returned so a best effort shutdown can be done.
func (s *Supervisor) pids() []int {
	pids := []int{}
	dirEntries, err := os.ReadDir(constants.DirProc)
	if err != nil {
		slog.Error("Unable to read directory", "directory", constants.DirProc, "error", err)
		return s.svcPIDs()
	}
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(dirEntry.Name())
		if err != nil {
			continue
		}
		statFile := filepath.Join(constants.DirProc, dirEntry.Name(), "stat")
		kt, err := isKernelThread(statFile)
		if err != nil {
			slog.Error("Unable to filter kernel thread", "pid", pid, "error", err)
			return s.svcPIDs()
		}
		if !kt {
			pids = append(pids, pid)
		}
	}
	return pids
}

// svcPIDs returns the PIDs of known services.
func (s *Supervisor) svcPIDs() []int {
	pids := []int{}
	for _, svc := range s.Services {
		pid := svc.PID()
		if pid != 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

func isKernelThread(statFile string) (bool, error) {
	const (
		flagsField  = 8
		nStatFields = 52
	)
	st, err := os.ReadFile(statFile)
	if err != nil {
		return false, fmt.Errorf("unable to read %s: %w", statFile, err)
	}
	fields := strings.Fields(string(st))
	if len(fields) != nStatFields {
		err = fmt.Errorf("expected %d fields in %s, got %d", nStatFields,
			statFile, len(fields))
		return false, err
	}
	statField := fields[flagsField]
	flags, err := strconv.Atoi(statField)
	if err != nil {
		return false, fmt.Errorf("unable to parse %s: %w", statFile, err)
	}
	return flags&PF_KTHREAD != 0, nil
}
