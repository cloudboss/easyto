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
	WaitInit()
	WaitStart()
	WaitStop() error
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
	ErrC     chan error
	InitC    chan struct{}
	StartC   chan struct{}
	optional bool
	shutdown bool
	cmd      exec.Cmd
}

func (s *svc) Start() error {
	if s.Init == nil {
		s.InitC <- struct{}{}
	} else {
		err := s.Init()
		s.InitC <- struct{}{}
		if err != nil {
			return err
		}
	}

	go func() {
		firstTime := true
		for {
			s.setCmd()

			if firstTime {
				slog.Info("Starting service", "service", s.cmd.Args)
				firstTime = false
			}

			s.cmd.Start()
			s.StartC <- struct{}{}

			err := s.cmd.Wait()
			if s.shutdown {
				s.ErrC <- err
				break
			}
			if err != nil {
				slog.Error("Process errored, will restart", "process", s.Args[0], "error", err)
			} else {
				slog.Warn("Process exited, will restart", "process", s.Args[0])
			}

			time.Sleep(5 * time.Second)
		}
	}()

	return nil
}

func (s *svc) WaitStop() error {
	return <-s.ErrC
}

func (s *svc) WaitInit() {
	<-s.InitC
}

func (s *svc) WaitStart() {
	<-s.StartC
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

func newSvc() svc {
	return svc{
		Dir:    "/",
		Env:    []string{},
		ErrC:   make(chan error, 1),
		InitC:  make(chan struct{}, 1),
		StartC: make(chan struct{}, 1),
	}
}

func (s *svc) setCmd() {
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
