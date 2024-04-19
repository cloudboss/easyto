package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudboss/easyto/preinit/constants"
	"github.com/cloudboss/easyto/preinit/login"
)

type SSHDService struct {
	svc
}

func NewSSHDService() Service {
	return &SSHDService{
		svc: svc{
			Args: []string{
				filepath.Join(constants.DirCB, "sshd"),
				"-D",
				"-f",
				filepath.Join(constants.DirCB, "sshd_config"),
				"-e",
			},
			Dir:  "/",
			Env:  []string{},
			Init: sshdInit,
			C:    make(chan error),
		},
	}
}

func sshdInit() error {
	fmt.Println("Initializing sshd")

	fmt.Println("Adding sshd privsep user")
	_, _, err := login.AddSystemUser(fs, SSHUser, SSHUser, SSHDir)
	if err != nil {
		fmt.Printf("Error adding sshd user: %s\n", err)
		return err
	}

	fmt.Println("Creating sshd privilege separation directory")
	if err := os.Mkdir(SSHDir, 0711); err != nil && !os.IsExist(err) {
		return fmt.Errorf("unable to create %s: %w", SSHDir, err)
	}

	fmt.Println("Creating RSA host key")
	rsaKeyPath := filepath.Join(constants.DirCB, "ssh_host_rsa_key")
	if _, err := os.Stat(rsaKeyPath); os.IsNotExist(err) {
		if err := sshKeygen("rsa", rsaKeyPath); err != nil {
			fmt.Printf("Error creating RSA host key: %+v\n", err)
			return err
		}
	}

	fmt.Println("Creating ED25519 host key")
	ed25519KeyPath := filepath.Join(constants.DirCB, "ssh_host_ed25519_key")
	if _, err := os.Stat(ed25519KeyPath); os.IsNotExist(err) {
		if err := sshKeygen("ed25519", ed25519KeyPath); err != nil {
			fmt.Printf("Error creating ED25519 host key: %+v\n", err)
			return err
		}
	}

	return nil
}

func sshKeygen(keyType, keyPath string) error {
	keygen := filepath.Join(constants.DirCB, "ssh-keygen")
	cmd := exec.Command(keygen, "-t", keyType, "-f", keyPath, "-N", "")
	return cmd.Run()
}
