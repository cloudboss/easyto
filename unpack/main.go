package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	cfg = &config{}
	cmd = cobra.Command{
		Use:   "unpack",
		Short: "A tool to convert a container image to an EC2 AMI",
		RunE: func(cmd *cobra.Command, args []string) error {
			packerArgs := []string{
				"build",
				"-var", fmt.Sprintf("archive_preinit=%s/preinit.tar", cfg.assetDir),
				"-var", fmt.Sprintf("ami_name=%s", cfg.amiName),
				"-var", fmt.Sprintf("archive_bootloader=%s/boot.tar", cfg.assetDir),
				"-var", fmt.Sprintf("archive_kernel=%s/kernel-%s.tar", cfg.assetDir,
					kernelVersion),
				"-var", fmt.Sprintf("exec_converter=%s/converter", cfg.assetDir),
				"-var", fmt.Sprintf("container_image=%s", cfg.containerImage),
				"-var", fmt.Sprintf("root_vol_size=%d", cfg.size),
				"-var", fmt.Sprintf("subnet_id=%s", cfg.subnetID),
				"build.pkr.hcl",
			}

			packer := exec.Command("./packer", packerArgs...)

			packer.Stdin = os.Stdin
			packer.Stdout = os.Stdout
			packer.Stderr = os.Stderr

			packer.Dir = cfg.packerDir

			packer.Env = append(os.Environ(), []string{
				"CHECKPOINT_DISABLE=1",
				fmt.Sprintf("PACKER_PLUGIN_PATH=%s/plugins", cfg.packerDir),
			}...)

			fmt.Printf("%+v\n", packer)

			cmd.SilenceUsage = true

			return packer.Run()
		},
	}
	// Value of kernelVersion is defined with ldflags.
	kernelVersion string
)

type config struct {
	amiName        string
	assetDir       string
	containerImage string
	packerDir      string
	size           int
	subnetID       string
}

func init() {
	this, err := os.Executable()
	if err != nil {
		panic(err)
	}
	assetDirRequired := false
	assetDir, err := filepath.Abs(filepath.Join(filepath.Dir(this), "..", "assets"))
	if err != nil {
		assetDirRequired = true
	}
	fmt.Printf("asset dir: %s\n", assetDir)

	packerDir, err := filepath.Abs(filepath.Join(filepath.Dir(this), "..", "packer"))
	if err != nil {
		panic(err)
	}
	cfg.packerDir = packerDir

	cmd.Flags().StringVarP(&cfg.amiName, "ami-name", "a", "", "Name of the AMI.")
	cmd.MarkFlagRequired("ami-name")

	cmd.Flags().StringVarP(&cfg.assetDir, "asset-directory", "A", assetDir,
		"Path to a directory containing asset files.")
	if assetDirRequired {
		cmd.MarkFlagRequired("asset-directory")
	}

	cmd.Flags().StringVarP(&cfg.containerImage, "container-image", "c", "",
		"Name of the container image.")
	cmd.MarkFlagRequired("container-image")

	cmd.Flags().IntVarP(&cfg.size, "size", "S", 2,
		"Size of the image root volume in GB.")

	cmd.Flags().StringVarP(&cfg.subnetID, "subnet-id", "s", "",
		"Name of the subnet in which to run the image builder.")
	cmd.MarkFlagRequired("subnet-id")
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
