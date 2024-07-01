package service

import (
	"log/slog"
)

type Main struct {
	svc
}

func NewMainService(command, env []string, workingDir string, uid, gid uint32) Service {
	svc := newSvc()
	svc.Args = command
	svc.Dir = workingDir
	svc.Env = env
	svc.GID = gid
	svc.UID = uid

	return &Main{svc: svc}
}

func (m *Main) Start() error {
	m.InitC <- struct{}{}
	m.setCmd()

	slog.Info("Starting main command", "command", m.cmd.Args)

	go func() {
		m.cmd.Start()
		m.StartC <- struct{}{}
		m.ErrC <- m.cmd.Wait()
	}()

	return nil
}
