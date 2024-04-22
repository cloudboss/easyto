package service

import (
	"fmt"
)

type Main struct {
	svc
}

func NewMainService(command, env []string, workingDir string, uid, gid uint32) Service {
	return &Main{
		svc: svc{
			Args: command,
			Dir:  workingDir,
			Env:  env,
			GID:  gid,
			UID:  uid,
			C:    make(chan error, 1),
		},
	}
}

func (m *Main) Start() error {
	m.init()

	fmt.Printf("Starting main command %+v\n", m.cmd)

	go func() {
		m.C <- m.cmd.Run()
	}()

	return nil
}
