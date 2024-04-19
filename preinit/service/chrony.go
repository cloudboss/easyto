package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudboss/easyto/preinit/constants"
	"github.com/cloudboss/easyto/preinit/login"
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

	chronyRunPath := filepath.Join(constants.DirRun, "chrony")
	err := os.Mkdir(chronyRunPath, 0750)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("unable to create %s: %w", chronyRunPath, err)
	}

	return nil
}
