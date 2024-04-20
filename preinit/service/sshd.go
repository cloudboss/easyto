package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/cloudboss/easyto/preinit/aws"
	"github.com/cloudboss/easyto/preinit/constants"
	"github.com/cloudboss/easyto/preinit/login"
	"github.com/spf13/afero"
)

const (
	loginUser = "cloudboss"
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
			Dir:      "/",
			Env:      []string{},
			Init:     sshdInit,
			C:        make(chan error),
			optional: true,
		},
	}
}

func sshdInit() error {
	fmt.Println("Initializing sshd")

	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	fmt.Println("Adding login user")
	loginHome := filepath.Join(constants.DirHome, loginUser)
	uid, gid, err := login.AddLoginUser(fs, loginUser, loginUser, loginHome)
	if err != nil {
		return fmt.Errorf("unable to add login user %s: %w\n", loginUser, err)
	}

	fmt.Println("Writing ssh public key for login user")
	sshDir := filepath.Join(loginHome, ".ssh")
	err = sshWritePubKey(fs, sshDir, uint16(uid), uint16(gid))
	if err != nil {
		return fmt.Errorf("unable to write SSH public key: %w", err)
	}

	fmt.Println("Adding sshd privsep user")
	_, _, err = login.AddSystemUser(fs, SSHUser, SSHUser, SSHDir)
	if err != nil {
		return fmt.Errorf("unable to add sshd privsep user %s: %w\n", SSHUser, err)
	}

	fmt.Println("Creating sshd privilege separation directory")
	if err := fs.Mkdir(SSHDir, 0711); err != nil && !os.IsExist(err) {
		return fmt.Errorf("unable to create %s: %w", SSHDir, err)
	}

	fmt.Println("Creating RSA host key")
	rsaKeyPath := filepath.Join(constants.DirCB, "ssh_host_rsa_key")
	if _, err := fs.Stat(rsaKeyPath); os.IsNotExist(err) {
		if err := sshKeygen("rsa", rsaKeyPath); err != nil {
			return fmt.Errorf("unable to create RSA host key: %+v\n", err)
		}
	}

	fmt.Println("Creating ED25519 host key")
	ed25519KeyPath := filepath.Join(constants.DirCB, "ssh_host_ed25519_key")
	if _, err := fs.Stat(ed25519KeyPath); os.IsNotExist(err) {
		if err := sshKeygen("ed25519", ed25519KeyPath); err != nil {
			return fmt.Errorf("unable to create ED25519 host key: %+v\n", err)
		}
	}

	return nil
}

func sshKeygen(keyType, keyPath string) error {
	keygen := filepath.Join(constants.DirCB, "ssh-keygen")
	cmd := exec.Command(keygen, "-t", keyType, "-f", keyPath, "-N", "")
	return cmd.Run()
}

func sshWritePubKey(fs afero.Fs, dir string, uid, gid uint16) error {
	pubKey, err := aws.GetSSHPubKey()
	if err != nil {
		return fmt.Errorf("unable to get SSH key from instance metadata: %w", err)
	}

	keyPath := filepath.Join(dir, "authorized_keys")

	f, err := fs.Create(keyPath)
	if err != nil {
		return fmt.Errorf("unable to create %s: %w", keyPath, err)
	}
	defer f.Close()

	err = fs.Chown(keyPath, int(uid), int(gid))
	if err != nil {
		return fmt.Errorf("unable to change ownership of %s: %w", keyPath, err)
	}

	err = fs.Chmod(keyPath, 0640)
	if err != nil {
		return fmt.Errorf("unable to change permissions of %s: %w", keyPath, err)
	}

	_, err = f.Write([]byte(pubKey))
	if err != nil {
		return fmt.Errorf("unable to write key to %s: %w", keyPath, err)
	}

	return nil
}
