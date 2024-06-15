package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/cloudboss/easyto/pkg/ctr2disk"
	"github.com/spf13/cobra"
)

var (
	cfg = config{}
	cmd = &cobra.Command{
		Use:   "ctr2disk",
		Short: "Convert a container image to a disk image",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			builder, err := ctr2disk.NewBuilder(
				ctr2disk.WithAssetDir(cfg.assetDir),
				ctr2disk.WithCTRImageName(cfg.image),
				ctr2disk.WithVMImageDevice(cfg.vmImageDevice),
				ctr2disk.WithVMImageMount(cfg.vmImageMount),
				ctr2disk.WithServices(cfg.services),
				ctr2disk.WithLoginUser(cfg.loginUser),
				ctr2disk.WithLoginShell(cfg.loginShell),
			)
			if err != nil {
				return fmt.Errorf("failed to create VM image builder: %w", err)
			}

			err = builder.MakeVMImage()
			if err != nil {
				return fmt.Errorf("failed to convert container image to VM image: %w", err)
			}

			return nil
		},
	}
)

type config struct {
	assetDir      string
	image         string
	vmImageDevice string
	vmImageMount  string
	services      []string
	loginUser     string
	loginShell    string
}

func init() {
	cmd.Flags().StringVarP(&cfg.assetDir, "asset-dir", "a", "",
		"Path to a directory containing asset files.")
	cmd.MarkFlagRequired("asset-dir")

	cmd.Flags().StringVarP(&cfg.image, "container-image", "i", "", "Container image to convert.")
	cmd.MarkFlagRequired("container-image")

	cmd.Flags().StringVarP(&cfg.vmImageDevice, "vm-image-device", "d", "",
		"Device on which VM image will be created.")
	cmd.MarkFlagRequired("vm-image-device")

	cmd.Flags().StringVarP(&cfg.vmImageMount, "vm-image-mount", "m", "/mnt",
		"Remote directory on which VM image device will be mounted.")

	cmd.Flags().StringSliceVarP(&cfg.services, "services", "s", []string{"chrony"},
		"Comma separated list of services to enable [chrony,ssh].")

	cmd.Flags().StringVar(&cfg.loginUser, "login-user", "cloudboss",
		"Login user to create in the VM image if ssh service is enabled.")

	loginShell := filepath.Join(constants.DirCB, "sh")
	cmd.Flags().StringVar(&cfg.loginShell, "login-shell", loginShell,
		"Login shell to use for the login user if ssh service is enabled.")
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
