package vmspec

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type VMSpec struct {
	Args        []string        `json:"args,omitempty"`
	Command     []string        `json:"command,omitempty"`
	Debug       bool            `json:"debug,omitempty"`
	Env         NameValueSource `json:"env,omitempty"`
	EnvFrom     EnvFromSource   `json:"env-from,omitempty"`
	ReplaceInit bool            `json:"replace-init,omitempty"`
	Security    SecurityContext `json:"security,omitempty"`
	Sysctls     NameValueSource `json:"sysctls,omitempty"`
	Volumes     Volumes         `json:"volumes,omitempty"`
	WorkingDir  string          `json:"working-dir,omitempty"`
}

func (v *VMSpec) Merge(other *VMSpec) *VMSpec {
	newVMSpec := v

	if other.Args != nil {
		newVMSpec.Args = other.Args
	}
	if other.Command != nil {
		newVMSpec.Command = other.Command
		// Always wipe the args from the image if the command is overridden.
		newVMSpec.Args = other.Args
	}

	if other.Debug {
		newVMSpec.Debug = other.Debug
	}

	if other.ReplaceInit {
		newVMSpec.ReplaceInit = other.ReplaceInit
	}

	newVMSpec.Env = newVMSpec.Env.Merge(other.Env)

	newVMSpec.Security = newVMSpec.Security.Merge(other.Security)

	if len(other.WorkingDir) != 0 {
		newVMSpec.WorkingDir = other.WorkingDir
	}

	if other.Volumes != nil {
		newVMSpec.Volumes = other.Volumes
	}

	if other.EnvFrom != nil {
		newVMSpec.EnvFrom = other.EnvFrom
	}

	if other.Sysctls != nil {
		newVMSpec.Sysctls = other.Sysctls
	}

	return newVMSpec
}

func (v *VMSpec) Validate() error {
	var errs error
	for _, ef := range v.EnvFrom {
		errs = errors.Join(errs, ef.Validate())
	}
	for _, ef := range v.Volumes {
		errs = errors.Join(errs, ef.Validate())
	}
	return errs
}

type NameValue struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type NameValueSource []NameValue

// Find returns the value of the item at key with its index, or -1 if not found.
func (n NameValueSource) Find(key string) (string, int) {
	for i, item := range n {
		if item.Name == key {
			return item.Value, i
		}
	}
	return "", -1
}

// Merge will merge NameValues from other with its own NameValues, returning a new
// copy. Overridden values will come first in the returned copy.
func (n NameValueSource) Merge(other NameValueSource) NameValueSource {
	if other == nil {
		cp := n
		return cp
	}
	newItems := NameValueSource{}
	for _, item := range n {
		if _, j := other.Find(item.Name); j < 0 {
			newItems = append(newItems, NameValue{
				Name:  item.Name,
				Value: item.Value,
			})
		}
	}
	return append(newItems, other...)
}

func (n NameValueSource) ToStrings() []string {
	stringItems := make([]string, len(n))
	for i, item := range n {
		stringItems[i] = item.Name + "=" + item.Value
	}
	return stringItems
}

type EnvFromSource []EnvFrom

type EnvFrom struct {
	Prefix         string                   `json:"prefix,omitempty"`
	S3Object       *S3ObjectEnvSource       `json:"s3-object,omitempty"`
	SecretsManager *SecretsManagerEnvSource `json:"secrets-manager,omitempty"`
	SSMParameter   *SSMParameterEnvSource   `json:"ssm-parameter,omitempty"`
}

func (e *EnvFrom) Validate() error {
	envNames := []string{}
	if e.S3Object != nil {
		envNames = append(envNames, "s3-object")
	}
	if e.SecretsManager != nil {
		envNames = append(envNames, "secrets-manager")
	}
	if e.SSMParameter != nil {
		envNames = append(envNames, "ssm-parameter")
	}
	lenEnvNames := len(envNames)
	if lenEnvNames > 1 {
		return fmt.Errorf("expected 1 environment source, got %d: %s", lenEnvNames,
			strings.Join(envNames, ", "))
	}
	return nil
}

type S3ObjectEnvSource struct {
	Bucket   string `json:"bucket,omitempty"`
	Object   string `json:"object,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type SecretsManagerEnvSource struct {
	Name     string `json:"name,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type SSMParameterEnvSource struct {
	Path     string `json:"path,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type Volume struct {
	EBS            *EBSVolumeSource            `json:"ebs,omitempty"`
	SecretsManager *SecretsManagerVolumeSource `json:"secrets-manager,omitempty"`
	SSMParameter   *SSMParameterVolumeSource   `json:"ssm-parameter,omitempty"`
	S3             *S3VolumeSource             `json:"s3,omitempty"`
}

func (v *Volume) Validate() error {
	volumeNames := []string{}
	if v.EBS != nil {
		volumeNames = append(volumeNames, "ebs")
	}
	if v.SecretsManager != nil {
		volumeNames = append(volumeNames, "secrets-manager")
	}
	if v.SSMParameter != nil {
		volumeNames = append(volumeNames, "ssm-parameter")
	}
	if v.S3 != nil {
		volumeNames = append(volumeNames, "s3")
	}
	lenVolumeNames := len(volumeNames)
	if lenVolumeNames > 1 {
		return fmt.Errorf("expected 1 volume source, got %d: %s", lenVolumeNames,
			strings.Join(volumeNames, ", "))
	}
	return nil
}

type Volumes []Volume

func (v Volumes) MountPoints() []string {
	mountPoints := []string{}
	for _, v := range v {
		if v.EBS != nil {
			mountPoints = append(mountPoints, v.EBS.Mount.Directory)
		}
		if v.SecretsManager != nil {
			mountPoints = append(mountPoints, v.SecretsManager.Mount.Directory)
		}
		if v.SSMParameter != nil {
			mountPoints = append(mountPoints, v.SSMParameter.Mount.Directory)
		}
		if v.S3 != nil {
			mountPoints = append(mountPoints, v.S3.Mount.Directory)
		}
	}
	// Reverse sort the mountpoints so children are listed before their
	// parents, to make it easier to unmount them in the correct order.
	sort.Sort(sort.Reverse(sort.StringSlice(mountPoints)))
	return mountPoints
}

type EBSVolumeSource struct {
	Attach bool   `json:"attach,omitempty"`
	Device string `json:"device,omitempty"`
	FSType string `json:"fs-type,omitempty"`
	MakeFS bool   `json:"make-fs,omitempty"`
	Mount  Mount  `json:"mount,omitempty"`
}

type SecretsManagerVolumeSource struct {
	Name  string `json:"name,omitempty"`
	Mount Mount  `json:"mount,omitempty"`
}

type SSMParameterVolumeSource struct {
	Mount    Mount  `json:"mount,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Path     string `json:"path,omitempty"`
}

type S3VolumeSource struct {
	Bucket    string `json:"bucket,omitempty"`
	KeyPrefix string `json:"key-prefix,omitempty"`
	Mount     Mount  `json:"mount,omitempty"`
	Optional  bool   `json:"optional,omitempty"`
}

type Mount struct {
	Directory string   `json:"directory,omitempty"`
	GroupID   int      `json:"group-id,omitempty"`
	Mode      string   `json:"mode,omitempty"`
	Options   []string `json:"options,omitempty"`
	UserID    int      `json:"user-id,omitempty"`
}

type SecurityContext struct {
	ReadonlyRootFS bool `json:"readonly-root-fs,omitempty"`
	RunAsGroupID   int  `json:"run-as-group-id,omitempty"`
	RunAsUserID    int  `json:"run-as-user-id,omitempty"`
	SSHD           SSHD `json:"sshd,omitempty"`
}

func (s SecurityContext) Merge(other SecurityContext) SecurityContext {
	if other.ReadonlyRootFS {
		s.ReadonlyRootFS = other.ReadonlyRootFS
	}
	if other.RunAsGroupID != 0 {
		s.RunAsGroupID = other.RunAsGroupID
	}
	if other.RunAsUserID != 0 {
		s.RunAsUserID = other.RunAsUserID
	}
	if other.SSHD.Enable {
		s.SSHD.Enable = other.SSHD.Enable
	}
	return s
}

type SSHD struct {
	Enable bool `json:"enable,omitempty"`
}
