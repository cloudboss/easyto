package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/cloudboss/easyto/ctr2ami/ctr2ami"
	"github.com/spf13/cobra"
)

var (
	cfg = config{}
	cmd = &cobra.Command{
		Use:   "ctr2ami",
		Short: "Convert a container image to an EC2 AMI",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			cacheHome, err := expandPath(cfg.cacheHome)
			if err != nil {
				return fmt.Errorf("failed to expand cache directory: %w", err)
			}

			builder, err := ctr2ami.NewBuilder(
				ctr2ami.WithBootloaderPath(cfg.bootloaderPath),
				ctr2ami.WithCacheHome(cacheHome),
				ctr2ami.WithInitPath(cfg.initPath),
				ctr2ami.WithKernelPath(cfg.kernelPath),
			)
			if err != nil {
				return fmt.Errorf("failed to create VM image builder: %w", err)
			}

			err = builder.MakeRawVMImage(cfg.image)
			if err != nil {
				return fmt.Errorf("failed to convert container image to VM image: %w", err)
			}

			return nil
		},
	}
)

type config struct {
	bootloaderPath string
	cacheHome      string
	image          string
	initPath       string
	kernelPath     string
}

func init() {
	cmd.Flags().StringVarP(&cfg.bootloaderPath, "bootloader-path", "b", "",
		"Path to a tar file containing bootloader files.")
	cmd.MarkFlagRequired("bootloader-path")
	cmd.Flags().StringVarP(&cfg.cacheHome, "cache-home", "c", "~/.ctr2ami",
		"Directory in which to cache images.")
	cmd.Flags().StringVarP(&cfg.image, "container-image", "i", "", "Container image to convert.")
	cmd.MarkFlagRequired("container-image")
	cmd.Flags().StringVarP(&cfg.kernelPath, "kernel-path", "k", "",
		"Path to a tar file containing kernel and modules.")
	cmd.MarkFlagRequired("kernel-path")
	cmd.Flags().StringVarP(&cfg.initPath, "init-path", "", "", "Path to an init executable.")
	cmd.MarkFlagRequired("init-path")
}

func expandPath(pth string) (string, error) {
	if strings.HasPrefix(pth, "~") {
		me, err := user.Current()
		if err != nil {
			return "", err
		}
		fields := strings.Split(pth, string(filepath.Separator))
		newFields := []string{me.HomeDir}
		newFields = append(newFields, fields[1:]...)
		return filepath.Join(newFields...), nil
	}

	return pth, nil
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
