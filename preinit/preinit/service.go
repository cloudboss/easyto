package preinit

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type Supervisor struct {
	Services    []Service
	Timeout     time.Duration
	signalFuncs []SignalFunc
}

func (s *Supervisor) Start() error {
	for _, service := range s.Services {
		signalFunc, err := service.Start()
		if err != nil {
			return err
		}
		s.signalFuncs = append(s.signalFuncs, signalFunc)
	}
	return nil
}

func (s *Supervisor) Stop() {
	for _, signalFunc := range s.signalFuncs {
		signalFunc(syscall.SIGTERM)
	}
}

func (s *Supervisor) Kill() {
	for _, signalFunc := range s.signalFuncs {
		signalFunc(syscall.SIGKILL)
	}
}

func (s *Supervisor) Wait() {
	poweroffC := make(chan os.Signal, 1)
	signal.Notify(poweroffC, SIGPWRBTN)

	doneC := make(chan struct{}, 1)

	go func() {
		for {
			_, err := syscall.Wait4(-1, nil, 0, nil)
			if err == syscall.ECHILD {
				// All processes have exited.
				break
			}
			// A process has exited, so stop all services.
			s.Stop()
		}
		doneC <- struct{}{}
	}()

	forever := time.Duration(1<<63 - 1)
	stopped := false

	// Create a timeout with an unreachable duration, to
	// be adjusted when a shutdown signal is received.
	timeout := time.NewTimer(forever)

	for !stopped {
		select {
		case <-poweroffC:
			fmt.Println("Got poweroff signal")
			// Try a graceful shutdown first.
			s.Stop()
			timeout.Reset(s.Timeout)
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

type ServiceHandle chan struct{}
type SignalFunc func(os.Signal)

type Service struct {
	Args []string
	Dir  string
	Env  []string
	GID  uint32
	UID  uint32
}

func (s *Service) Start() (SignalFunc, error) {
	cmd := &exec.Cmd{
		Args:   s.Args,
		Dir:    s.Dir,
		Env:    s.Env,
		Path:   s.Args[0],
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		SysProcAttr: &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Gid: s.GID,
				Uid: s.UID,
			},
		},
	}

	fmt.Printf("Starting service %+v\n", cmd)

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	signalFunc := func(signal os.Signal) {
		cmd.Process.Signal(signal)
	}

	return signalFunc, nil
}
