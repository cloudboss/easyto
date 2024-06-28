package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/cloudboss/easyto/pkg/login"
)

type ChronyService struct {
	svc
}

func NewChronyService() Service {
	return &ChronyService{
		svc: svc{
			Args: []string{
				filepath.Join(constants.DirETSbin, "chronyd"),
				"-d",
			},
			Dir:  "/",
			Env:  []string{},
			Init: chronyInit,
			C:    make(chan error, 1),
		},
	}
}

func chronyInit() error {
	slog.Info("Initializing chrony")

	_, usersByName, _, err := login.ParsePasswd(fs, constants.FileEtcPasswd)
	if err != nil {
		return fmt.Errorf("unable to parse %s: %s", constants.FileEtcPasswd, err)
	}
	user, ok := usersByName[constants.ChronyUser]
	if !ok {
		return fmt.Errorf("user %s not found", constants.ChronyUser)
	}

	chronyRunPath := filepath.Join(constants.DirETRun, "chrony")
	err = os.Mkdir(chronyRunPath, 0750)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("unable to create %s: %w", chronyRunPath, err)
	}

	err = os.Chown(chronyRunPath, int(user.UID), int(user.GID))
	if err != nil {
		return fmt.Errorf("unable to change ownership of %s: %w", chronyRunPath, err)
	}

	return nil
}
