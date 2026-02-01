package tree

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/cloudboss/easyto/pkg/constants"
	"github.com/cloudboss/easyto/pkg/sourceami"
	"github.com/spf13/cobra"
)

var (
	amiCfg = &amiConfig{}
	AMICmd = &cobra.Command{
		Use:   "ami",
		Short: "Convert a container image to an EC2 AMI",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			assetDir, err := expandPath(amiCfg.assetDir)
			if err != nil {
				return fmt.Errorf("failed to expand asset directory path: %w", err)
			}
			if _, err = os.Stat(assetDir); os.IsNotExist(err) {
				return fmt.Errorf("asset directory does not exist: %s", assetDir)
			}
			amiCfg.assetDir = assetDir

			packerDir, err := expandPath(amiCfg.packerDir)
			if err != nil {
				return fmt.Errorf("failed to expand packer directory path: %w", err)
			}
			if _, err = os.Stat(packerDir); os.IsNotExist(err) {
				return fmt.Errorf("packer directory does not exist: %s", packerDir)
			}
			amiCfg.packerDir = packerDir

			svcErr := validateServices(amiCfg.services)
			sshErr := validateSSHInterface(amiCfg.sshInterface)
			modeErr := validateBuilderImageMode(amiCfg.builderImageMode, amiCfg.builderImage)
			return errors.Join(svcErr, sshErr, modeErr)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			resp, err := sourceami.Resolve(ctx, amiCfg.builderImage, constants.ETVersion)
			if err != nil {
				return fmt.Errorf("failed to resolve builder AMI: %w", err)
			}

			if amiCfg.builderImage != "" && amiCfg.builderImageMode == "" {
				fmt.Println("No --builder-image-mode specified, defaulting to slow mode")
			}

			if amiCfg.builderImageMode != "" {
				switch amiCfg.builderImageMode {
				case "fast":
					resp.Mode = sourceami.ModeFast
				case "slow":
					resp.Mode = sourceami.ModeSlow
				}
			}

			var packerSubdir, sshUsername string
			switch resp.Mode {
			case sourceami.ModeFast:
				packerSubdir = "fast"
				sshUsername = "cloudboss"
			case sourceami.ModeSlow:
				packerSubdir = "slow"
				sshUsername = "admin"
				if amiCfg.builderImage == "" {
					fmt.Printf("Builder AMI %s%s not found, falling back to slow mode\n",
						constants.AMIPatternCloudboss, constants.ETVersion)
				}
			}

			if amiCfg.builderImage != "" {
				sshUsername = amiCfg.builderImageLoginUser
			}

			if amiCfg.debug {
				fmt.Printf("Using %s path with AMI %s\n", packerSubdir, resp.AMI)
			}

			quotedServices := bytes.NewBufferString("")
			err = json.NewEncoder(quotedServices).Encode(amiCfg.services)
			if err != nil {
				return fmt.Errorf("unexpected value for services: %w", err)
			}

			packerArgs := []string{
				"build",
				"-var", fmt.Sprintf("ami_name=%s", amiCfg.amiName),
				"-var", fmt.Sprintf("container_image=%s", amiCfg.containerImage),
				"-var", fmt.Sprintf("debug=%t", amiCfg.debug),
				"-var", fmt.Sprintf("login_user=%s", amiCfg.loginUser),
				"-var", fmt.Sprintf("login_shell=%s", amiCfg.loginShell),
				"-var", fmt.Sprintf("root_device_name=%s", amiCfg.rootDeviceName),
				"-var", fmt.Sprintf("root_vol_size=%d", amiCfg.size),
				"-var", fmt.Sprintf("services=%s", quotedServices.String()),
				"-var", fmt.Sprintf("source_ami=%s", resp.AMI),
				"-var", fmt.Sprintf("ssh_interface=%s", amiCfg.sshInterface),
				"-var", fmt.Sprintf("ssh_username=%s", sshUsername),
				"-var", fmt.Sprintf("subnet_id=%s", amiCfg.subnetID),
			}

			if resp.Mode == sourceami.ModeSlow {
				packerArgs = append(packerArgs, "-var", fmt.Sprintf("asset_dir=%s", amiCfg.assetDir))
			}

			packerArgs = append(packerArgs, "build.pkr.hcl")

			packerWorkDir := filepath.Join(amiCfg.packerDir, packerSubdir)
			packer := exec.Command(filepath.Join(amiCfg.packerDir, "packer"), packerArgs...)

			packer.Stdin = os.Stdin
			packer.Stdout = os.Stdout
			packer.Stderr = os.Stderr

			packer.Dir = packerWorkDir

			packer.Env = append(os.Environ(), []string{
				"CHECKPOINT_DISABLE=1",
				fmt.Sprintf("PACKER_PLUGIN_PATH=%s/plugins", amiCfg.packerDir),
			}...)

			if amiCfg.debug {
				fmt.Printf("%+v\n", packer)
			}

			cmd.SilenceUsage = true

			return packer.Run()
		},
	}
)

type amiConfig struct {
	amiName               string
	assetDir              string
	builderImage          string
	builderImageLoginUser string
	builderImageMode      string
	containerImage        string
	debug                 bool
	loginUser             string
	loginShell            string
	packerDir             string
	rootDeviceName        string
	services              []string
	size                  int
	sshInterface          string
	subnetID              string
}

func init() {
	this, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get executable path: %s\n", err)
		os.Exit(1)
	}
	// In case we are a symlink, get the real path.
	realThis, err := filepath.EvalSymlinks(this)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get real path of executable: %s\n", err)
		os.Exit(1)
	}
	assetDir, err := filepath.Abs(filepath.Join(filepath.Dir(realThis), "..", "assets"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get absolute path of asset directory: %s\n", err)
		os.Exit(1)
	}
	packerDir, err := filepath.Abs(filepath.Join(filepath.Dir(realThis), "..", "packer"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get absolute path of packer directory: %s\n", err)
		os.Exit(1)
	}

	AMICmd.Flags().StringVarP(&amiCfg.amiName, "ami-name", "a", "", "Name of the AMI.")
	AMICmd.MarkFlagRequired("ami-name")

	AMICmd.Flags().StringVarP(&amiCfg.assetDir, "asset-directory", "A", assetDir,
		"Path to a directory containing asset files.")

	AMICmd.Flags().StringVar(&amiCfg.builderImage, "builder-image", "",
		"AMI ID or name pattern for the builder image. If not specified, uses the easyto builder AMI matching the current version, falling back to Debian.")

	AMICmd.Flags().StringVar(&amiCfg.builderImageLoginUser, "builder-image-login-user", "cloudboss",
		"SSH login user for the builder image when using --builder-image.")

	AMICmd.Flags().StringVar(&amiCfg.builderImageMode, "builder-image-mode", "",
		"Build mode to use with --builder-image. Must be 'fast' or 'slow'. Fast mode assumes easyto is pre-installed on the builder image.")

	AMICmd.Flags().StringVarP(&amiCfg.packerDir, "packer-directory", "P", packerDir,
		"Path to a directory containing packer and its configuration.")

	AMICmd.Flags().StringVarP(&amiCfg.containerImage, "container-image", "c", "",
		"Name of the container image.")
	AMICmd.MarkFlagRequired("container-image")

	AMICmd.Flags().IntVarP(&amiCfg.size, "size", "S", 10,
		"Size of the image root volume in GB.")

	AMICmd.Flags().StringVar(&amiCfg.loginUser, "login-user", "cloudboss",
		"Login user to create in the VM image if ssh service is enabled.")

	loginShell := filepath.Join(constants.DirETBin, "sh")
	AMICmd.Flags().StringVar(&amiCfg.loginShell, "login-shell", loginShell,
		"Shell to use for the login user if ssh service is enabled.")

	AMICmd.Flags().StringVar(&amiCfg.rootDeviceName, "root-device-name", "/dev/xvda",
		"Name of the AMI root device.")

	AMICmd.Flags().StringSliceVar(&amiCfg.services, "services", []string{"chrony"},
		"Comma separated list of services to enable [chrony,ssh]. Use an empty string to disable all services.")

	AMICmd.Flags().StringVarP(&amiCfg.sshInterface, "ssh-interface", "i", "public_ip",
		"The interface for ssh connection to the builder. Must be one of 'public_ip' or 'private_ip'.")

	AMICmd.Flags().StringVarP(&amiCfg.subnetID, "subnet-id", "s", "",
		"ID of the subnet in which to run the image builder.")
	AMICmd.MarkFlagRequired("subnet-id")

	AMICmd.Flags().BoolVar(&amiCfg.debug, "debug", false, "Enable debug output.")
}

func expandPath(pth string) (string, error) {
	expanded := pth
	if strings.HasPrefix(pth, "~/") {
		me, err := user.Current()
		if err != nil {
			return "", err
		}
		fields := strings.Split(pth, string(filepath.Separator))
		newFields := []string{me.HomeDir}
		newFields = append(newFields, fields[1:]...)
		expanded = filepath.Join(newFields...)
	}

	return filepath.Abs(expanded)
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

func validateSSHInterface(sshInterface string) error {
	switch sshInterface {
	case "public_ip", "private_ip":
		return nil
	default:
		return fmt.Errorf("invalid ssh interface %s", sshInterface)
	}
}

func validateBuilderImageMode(mode, builderImage string) error {
	if mode == "" {
		return nil
	}
	if builderImage == "" {
		return fmt.Errorf("--builder-image-mode requires --builder-image")
	}
	switch mode {
	case "fast", "slow":
		return nil
	default:
		return fmt.Errorf("invalid builder image mode %s, must be 'fast' or 'slow'", mode)
	}
}
