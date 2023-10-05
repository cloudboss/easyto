package preinit

import (
	"errors"
	"fmt"
	"strings"
)

type VMSpec struct {
	Args        []string        `json:"args,omitempty"`
	Command     []string        `json:"command,omitempty"`
	Env         NameValueSource `json:"env,omitempty"`
	EnvFrom     EnvFromSource   `json:"env-from,omitempty"`
	ReplaceInit bool            `json:"replace-init,omitempty"`
	Security    SecurityContext `json:"security,omitempty"`
	Sysctls     NameValueSource `json:"sysctls,omitempty"`
	Volumes     []Volume        `json:"volumes,omitempty"`
	WorkingDir  string          `json:"working-dir,omitempty"`
}

func (v *VMSpec) merge(other *VMSpec) *VMSpec {
	newVMSpec := v

	if other.Args != nil {
		newVMSpec.Args = other.Args
	}
	if other.Command != nil {
		newVMSpec.Command = other.Command
		// Always wipe the args from the image if the command is overridden.
		newVMSpec.Args = other.Args
	}

	newVMSpec.Env = newVMSpec.Env.merge(other.Env)

	if other.Security.ReadonlyRootFS {
		newVMSpec.Security.ReadonlyRootFS = other.Security.ReadonlyRootFS
	}
	if other.Security.RunAsGroupID != 0 {
		newVMSpec.Security.RunAsGroupID = other.Security.RunAsGroupID
	}
	if other.Security.RunAsUserID != 0 {
		newVMSpec.Security.RunAsUserID = other.Security.RunAsUserID
	}

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
	return errs
}

type NameValue struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type NameValueSource []NameValue

// find returns the value of the item at key with its index, or -1 if not found.
func (n NameValueSource) find(key string) (string, int) {
	for i, item := range n {
		if item.Name == key {
			return item.Value, i
		}
	}
	return "", -1
}

// merge will merge NameValues from other with its own NameValues, returning a new
// copy. Overridden values will come first in the returned copy.
func (n NameValueSource) merge(other NameValueSource) NameValueSource {
	if other == nil {
		cp := n
		return cp
	}
	newItems := NameValueSource{}
	for _, item := range n {
		if _, j := other.find(item.Name); j < 0 {
			newItems = append(newItems, NameValue{
				Name:  item.Name,
				Value: item.Value,
			})
		}
	}
	return append(newItems, other...)
}

func (n NameValueSource) toStrings() []string {
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
}

type EBSVolumeSource struct {
	Attach bool   `json:"attach,omitempty"`
	Device string `json:"device,omitempty"`
	FSType string `json:"fs-type,omitempty"`
	MakeFS bool   `json:"make-fs,omitempty"`
	Mount  Mount  `json:"mount,omitempty"`
}

type SecretsManagerVolumeSource struct {
	Name       string `json:"name,omitempty"`
	MountPoint Mount  `json:"mount-point,omitempty"`
}

type SSMParameterVolumeSource struct {
	Mount    Mount  `json:"mount,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Path     string `json:"path,omitempty"`
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
}
