package service

import (
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/spf13/afero"
)

var (
	fs = afero.NewOsFs()
)

type Service interface {
	Start() error
	Wait() error
	Stop()
	Kill()
	Optional() bool
}

type InitFunc func() error

type svc struct {
	Args     []string
	Dir      string
	Env      []string
	GID      uint32
	UID      uint32
	Init     InitFunc
	C        chan error
	optional bool
	shutdown bool
	cmd      exec.Cmd
}

func (s *svc) Start() error {
	if s.Init != nil {
		err := s.Init()
		if err != nil {
			return err
		}
	}

	s.init()

	slog.Info("Starting service", "service", s.cmd)

	go func() {
		for {
			err := s.cmd.Run()
			if !s.shutdown {
				if err != nil {
					slog.Error("Process errored, will restart", "process", s.Args[0], "error", err)
				} else {
					slog.Warn("Process exited, will restart", "process", s.Args[0])
				}
				s.init()
				time.Sleep(5 * time.Second)
			} else {
				s.C <- err
				break
			}
		}
	}()

	return nil
}

func (s *svc) Wait() error {
	return <-s.C
}

func (s *svc) Stop() {
	s.signal(syscall.SIGTERM)
}

func (s *svc) Kill() {
	s.signal(syscall.SIGKILL)
}

func (s *svc) Optional() bool {
	return s.optional
}

func (s *svc) signal(signal os.Signal) {
	if s.cmd.Process != nil {
		slog.Debug("Sending signal", "signal", signal, "command", s.Args[0])
		s.shutdown = true
		s.cmd.Process.Signal(signal)
	} else {
		s.C <- nil
	}
}

func (s *svc) init() {
	s.cmd = exec.Cmd{
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
}
