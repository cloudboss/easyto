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
	Optional() bool
	PID() int
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
			if s.shutdown {
				s.C <- err
				break
			}
			if err != nil {
				slog.Error("Process errored, will restart", "process", s.Args[0], "error", err)
			} else {
				slog.Warn("Process exited, will restart", "process", s.Args[0])
			}
			s.init()
			time.Sleep(5 * time.Second)
		}
	}()

	return nil
}

func (s *svc) Wait() error {
	return <-s.C
}

func (s *svc) Stop() {
	s.shutdown = true
}

func (s *svc) Optional() bool {
	return s.optional
}

func (s *svc) PID() int {
	if s.cmd.Process != nil {
		return s.cmd.Process.Pid
	}
	return 0
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
