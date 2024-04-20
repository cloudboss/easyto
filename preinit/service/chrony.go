package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudboss/easyto/lib/constants"
	"github.com/cloudboss/easyto/lib/login"
)

type ChronyService struct {
	svc
}

func NewChronyService() Service {
	return &ChronyService{
		svc: svc{
			Args: []string{
				filepath.Join(constants.DirCB, "chronyd"),
				"-d",
				"-u",
				ChronyUser,
			},
			Dir:  "/",
			Env:  []string{},
			Init: chronyInit,
			C:    make(chan error),
		},
	}
}

func chronyInit() error {
	fmt.Println("Initializing chrony")

	fmt.Println("Adding chrony user")
	uid, gid, err := login.AddSystemUser(fs, ChronyUser, ChronyUser, "/nonexistent")
	if err != nil {
		fmt.Printf("Error adding chrony user: %s\n", err)
		return err
	}

	chronyRunPath := filepath.Join(constants.DirRun, "chrony")
	err = os.Mkdir(chronyRunPath, 0750)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("unable to create %s: %w", chronyRunPath, err)
	}

	err = os.Chown(chronyRunPath, int(uid), int(gid))
	if err != nil {
		return fmt.Errorf("unable to change ownership of %s: %w", chronyRunPath, err)
	}

	return nil
}
