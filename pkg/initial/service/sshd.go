package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/cloudboss/easyto/pkg/initial/aws"
	"github.com/cloudboss/easyto/pkg/login"
	"github.com/spf13/afero"
)

type SSHDService struct {
	svc
}

func NewSSHDService() Service {
	return &SSHDService{
		svc: svc{
			Args: []string{
				filepath.Join(constants.DirETSbin, "sshd"),
				"-D",
				"-f",
				filepath.Join(constants.DirETEtc, "ssh", "sshd_config"),
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
	slog.Info("Initializing sshd")

	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	loginUser, err := getLoginUser(fs)
	if err != nil {
		return fmt.Errorf("unable to get login user: %w", err)
	}

	_, userByName, _, err := login.ParsePasswd(fs, constants.FileEtcPasswd)
	if err != nil {
		return fmt.Errorf("unable to parse %s: %s\n", constants.FileEtcPasswd, err)
	}
	user, ok := userByName[loginUser]
	if !ok {
		return fmt.Errorf("login user %s not found in %s", loginUser,
			constants.FileEtcPasswd)
	}

	slog.Debug("Writing ssh public key", "user", loginUser)
	sshDir := filepath.Join(user.HomeDir, ".ssh")
	err = sshWritePubKey(fs, sshDir, user.UID, user.GID)
	if err != nil {
		return fmt.Errorf("unable to write SSH public key: %w", err)
	}

	slog.Debug("Creating RSA host key")
	rsaKeyPath := filepath.Join(constants.DirETEtc, "ssh", "ssh_host_rsa_key")
	if _, err := fs.Stat(rsaKeyPath); os.IsNotExist(err) {
		if err := sshKeygen("rsa", rsaKeyPath); err != nil {
			return fmt.Errorf("unable to create RSA host key: %w", err)
		}
	}

	slog.Debug("Creating ED25519 host key")
	ed25519KeyPath := filepath.Join(constants.DirETEtc, "ssh", "ssh_host_ed25519_key")
	if _, err := fs.Stat(ed25519KeyPath); os.IsNotExist(err) {
		if err := sshKeygen("ed25519", ed25519KeyPath); err != nil {
			return fmt.Errorf("unable to create ED25519 host key: %w", err)
		}
	}

	return nil
}

func sshKeygen(keyType, keyPath string) error {
	keygen := filepath.Join(constants.DirETBin, "ssh-keygen")
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

// getLoginUser returns the login username for the system. If the image
// was built with easyto and sshd is enabled, this should be the name of
// the one directory under the easyto home directory.
func getLoginUser(fs afero.Fs) (string, error) {
	entries, err := afero.ReadDir(fs, constants.DirETHome)
	if err != nil {
		return "", err
	}
	if len(entries) != 1 {
		return "", fmt.Errorf("expected one entry in %s", constants.DirETHome)
	}

	return entries[0].Name(), nil
}
