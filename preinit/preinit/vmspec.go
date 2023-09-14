package preinit

import (
	"errors"
	"fmt"
)

type VMSpec struct {
	Args       []string        `json:"cmd,omitempty"`
	Command    []string        `json:"entrypoint,omitempty"`
	Env        EnvVarSource    `json:"env,omitempty"`
	EnvFrom    EnvFromSource   `json:"env-from,omitempty"`
	Security   SecurityContext `json:"security,omitempty"`
	Volumes    []Volume        `json:"volumes,omitempty"`
	WorkingDir string          `json:"working-dir,omitempty"`
}

func (v *VMSpec) merge(other *VMSpec) *VMSpec {
	newVMSpec := v

	if other.Args != nil {
		newVMSpec.Args = other.Args
	}
	if other.Command != nil {
		newVMSpec.Command = other.Command
	}

	newVMSpec.Env = newVMSpec.Env.merge(other.Env)

	if other.Security.ReadonlyRootFS {
		newVMSpec.Security.ReadonlyRootFS = other.Security.ReadonlyRootFS
	}
	if other.Security.RunAsGroupID != 0 {
		newVMSpec.Security.RunAsGroupID = other.Security.RunAsGroupID
	}
	if other.Security.RunAsGroupID != 0 {
		newVMSpec.Security.RunAsGroupID = other.Security.RunAsGroupID
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

	return newVMSpec
}

func (v *VMSpec) Validate() error {
	var errs error
	for _, ef := range v.EnvFrom {
		errs = errors.Join(errs, ef.Validate())
	}
	return errs
}

type EnvVar struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type EnvVarSource []EnvVar

// find returns the value of the environment variable key with its index,
// or -1 if not found.
func (e EnvVarSource) find(key string) (string, int) {
	for i, envVar := range e {
		if envVar.Name == key {
			return envVar.Value, i
		}
	}
	return "", -1
}

// merge will merge EnvVars from other with its own EnvVars, returning a new
// copy. Overridden values will come first in the returned copy.
func (e EnvVarSource) merge(other EnvVarSource) EnvVarSource {
	if other == nil {
		cp := e
		return cp
	}
	newEnvVars := EnvVarSource{}
	for _, envVar := range e {
		if _, j := other.find(envVar.Name); j < 0 {
			newEnvVars = append(newEnvVars, EnvVar{
				Name:  envVar.Name,
				Value: envVar.Value,
			})
		}
	}
	return append(newEnvVars, other...)
}

func (e EnvVarSource) toStrings() []string {
	stringEnv := make([]string, len(e))
	for i, envVar := range e {
		stringEnv[i] = envVar.Name + "=" + envVar.Value
	}
	return stringEnv
}

type EnvFromSource []EnvFrom

type EnvFrom struct {
	Prefix         string                   `json:"prefix,omitempty"`
	S3Object       *S3ObjectEnvSource       `json:"s3-object,omitempty"`
	SecretsManager *SecretsManagerEnvSource `json:"secrets-manager,omitempty"`
	SSMParameter   *SSMParameterEnvSource   `json:"ssm-parameter,omitempty"`
}

func (e *EnvFrom) Validate() error {
	nonNils := 0
	if e.S3Object != nil {
		nonNils++
	}
	if e.SecretsManager != nil {
		nonNils++
	}
	if e.SSMParameter != nil {
		nonNils++
	}
	if nonNils != 1 {
		return fmt.Errorf("expected 1 environment source, got %d", nonNils)
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

type Mount struct {
	Directory string   `json:"directory,omitempty"`
	GroupID   int      `json:"group,omitempty"`
	Mode      string   `json:"mode,omitempty"`
	Options   []string `json:"options,omitempty"`
	UserID    int      `json:"owner,omitempty"`
}

type SecurityContext struct {
	ReadonlyRootFS bool `json:"readonly-root-fs,omitempty"`
	RunAsGroupID   int  `json:"run-as-group-id,omitempty"`
	RunAsUserID    int  `json:"run-as-user-id,omitempty"`
}
