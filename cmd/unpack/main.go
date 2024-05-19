package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cfg = &config{}
	cmd = cobra.Command{
		Use:   "unpack",
		Short: "A tool to convert a container image to an EC2 AMI",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			assetDir, err := expandPath(cfg.assetDir)
			if err != nil {
				return fmt.Errorf("failed to expand asset directory path: %w", err)
			}
			cfg.assetDir = assetDir

			return validateServices(cfg.services)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			quotedServices := bytes.NewBufferString("")
			err := json.NewEncoder(quotedServices).Encode(cfg.services)
			if err != nil {
				// Unlikely that []string cannot be encoded to JSON, but check anyway.
				return fmt.Errorf("unexpected value for services: %w", err)
			}

			packerArgs := []string{
				"build",
				"-var", fmt.Sprintf("ami_name=%s", cfg.amiName),
				"-var", fmt.Sprintf("asset_dir=%s", cfg.assetDir),
				"-var", fmt.Sprintf("container_image=%s", cfg.containerImage),
				"-var", fmt.Sprintf("login_user=%s", cfg.loginUser),
				"-var", fmt.Sprintf("login_shell=%s", cfg.loginShell),
				"-var", fmt.Sprintf("root_device_name=%s", cfg.rootDeviceName),
				"-var", fmt.Sprintf("root_vol_size=%d", cfg.size),
				"-var", fmt.Sprintf("services=%s", quotedServices.String()),
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
)

type config struct {
	amiName        string
	assetDir       string
	containerImage string
	loginUser      string
	loginShell     string
	packerDir      string
	rootDeviceName string
	services       []string
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

	cmd.Flags().StringVar(&cfg.loginUser, "login-user", "cloudboss",
		"Login user to create in the VM image if ssh service is enabled.")

	cmd.Flags().StringVar(&cfg.loginShell, "login-shell", "/bin/sh",
		"Shell to use for the login user if ssh service is enabled.")

	cmd.Flags().StringVar(&cfg.rootDeviceName, "root-device-name", "/dev/xvda",
		"Name of the AMI root device.")

	cmd.Flags().StringVarP(&cfg.subnetID, "subnet-id", "s", "",
		"Name of the subnet in which to run the image builder.")
	cmd.MarkFlagRequired("subnet-id")

	cmd.Flags().StringSliceVar(&cfg.services, "services", []string{"chrony"},
		"Comma separated list of services to enable [chrony,ssh].")
}

func validateServices(services []string) error {
	for _, svc := range services {
		switch svc {
		case "chrony", "ssh":
			continue
		default:
			return fmt.Errorf("invalid service %s", svc)
		}
	}

	return nil
}

func expandPath(pth string) (string, error) {
	if strings.HasPrefix(pth, "~/") {
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
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
