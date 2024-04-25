package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/cloudboss/easyto/lib/constants"
	"github.com/cloudboss/easyto/lib/login"
	"github.com/cloudboss/easyto/preinit/aws"
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
			C:        make(chan error, 1),
			optional: true,
		},
	}
}

func sshdInit() error {
	fmt.Println("Initializing sshd")

	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	_, userByName, err := login.ParsePasswd(fs, constants.FileEtcPasswd)
	if err != nil {
		return fmt.Errorf("unable to parse %s: %s\n", constants.FileEtcPasswd, err)
	}
	user, ok := userByName["cloudboss"]
	if !ok {
		return fmt.Errorf("login user not found")
	}

	fmt.Println("Writing ssh public key for login user")
	sshDir := filepath.Join(user.HomeDir, ".ssh")
	err = sshWritePubKey(fs, sshDir, user.UID, user.GID)
	if err != nil {
		return fmt.Errorf("unable to write SSH public key: %w", err)
	}

	fmt.Println("Creating RSA host key")
	rsaKeyPath := filepath.Join(constants.DirCB, "ssh_host_rsa_key")
	if _, err := fs.Stat(rsaKeyPath); os.IsNotExist(err) {
		if err := sshKeygen("rsa", rsaKeyPath); err != nil {
			return fmt.Errorf("unable to create RSA host key: %w", err)
		}
	}

	fmt.Println("Creating ED25519 host key")
	ed25519KeyPath := filepath.Join(constants.DirCB, "ssh_host_ed25519_key")
	if _, err := fs.Stat(ed25519KeyPath); os.IsNotExist(err) {
		if err := sshKeygen("ed25519", ed25519KeyPath); err != nil {
			return fmt.Errorf("unable to create ED25519 host key: %w", err)
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
