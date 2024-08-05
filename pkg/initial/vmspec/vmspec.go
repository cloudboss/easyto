package vmspec

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"dario.cat/mergo"
)

const (
	pathEnvDefault = "/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin"
)

var DefaultServices = []string{"chrony", "ssh"}

type VMSpec struct {
	Args                []string        `json:"args,omitempty"`
	Command             []string        `json:"command,omitempty"`
	Debug               bool            `json:"debug,omitempty"`
	DisableServices     []string        `json:"disable-services,omitempty"`
	Env                 NameValueSource `json:"env,omitempty"`
	EnvFrom             EnvFromSource   `json:"env-from,omitempty"`
	InitScripts         []string        `json:"init-scripts,omitempty"`
	ReplaceInit         bool            `json:"replace-init,omitempty"`
	Security            SecurityContext `json:"security,omitempty"`
	ShutdownGracePeriod int             `json:"shutdown-grace-period,omitempty"`
	Sysctls             NameValueSource `json:"sysctls,omitempty"`
	Volumes             Volumes         `json:"volumes,omitempty"`
	WorkingDir          string          `json:"working-dir,omitempty"`
}

func (v *VMSpec) Merge(other *VMSpec) error {
	err := mergo.Merge(v, other, mergo.WithOverride, mergo.WithoutDereference,
		mergo.WithTransformers(nameValueTransformer{}))
	if err != nil {
		return err
	}
	if other.Command != nil {
		// Override args if command is set, even if zero value.
		v.Args = other.Args
	}
	v.SetDefaults()
	return nil
}

func (v *VMSpec) SetDefaults() {
	_, i := v.Env.Find("PATH")
	if i < 0 {
		pathEnv := NameValue{Name: "PATH", Value: pathEnvDefault}
		v.Env = append(v.Env, pathEnv)
	}
	if v.Security.RunAsGroupID == nil {
		v.Security.RunAsGroupID = p(0)
	}
	if v.Security.RunAsUserID == nil {
		v.Security.RunAsUserID = p(0)
	}
	for _, volume := range v.Volumes {
		if volume.EBS != nil {
			if volume.EBS.Mount.GroupID == nil {
				volume.EBS.Mount.GroupID = v.Security.RunAsGroupID
			}
			if volume.EBS.Mount.UserID == nil {
				volume.EBS.Mount.UserID = v.Security.RunAsUserID
			}
		}
		if volume.SecretsManager != nil {
			if volume.SecretsManager.Mount.GroupID == nil {
				volume.SecretsManager.Mount.GroupID = v.Security.RunAsGroupID
			}
			if volume.SecretsManager.Mount.UserID == nil {
				volume.SecretsManager.Mount.UserID = v.Security.RunAsUserID
			}
		}
		if volume.SSM != nil {
			if volume.SSM.Mount.GroupID == nil {
				volume.SSM.Mount.GroupID = v.Security.RunAsGroupID
			}
			if volume.SSM.Mount.UserID == nil {
				volume.SSM.Mount.UserID = v.Security.RunAsUserID
			}
		}
		if volume.S3 != nil {
			if volume.S3.Mount.GroupID == nil {
				volume.S3.Mount.GroupID = v.Security.RunAsGroupID
			}
			if volume.S3.Mount.UserID == nil {
				volume.S3.Mount.UserID = v.Security.RunAsUserID
			}
		}
	}
}

func (v *VMSpec) Validate() error {
	var errs error
	errs = errors.Join(errs, ValidateServices(v.DisableServices))
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

// ToMap converts a NameValueSource to a map[string]string.
func (n NameValueSource) ToMap() map[string]string {
	m := map[string]string{}
	for _, item := range n {
		m[item.Name] = item.Value
	}
	return m
}

type nameValueTransformer struct{}

// Transformer merges NameValueSource types. Values from src override values from dst if
// both have the same Name. Items in src with Name not existing in dst are appended to dst.
func (n nameValueTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	nvType := reflect.TypeOf(NameValueSource{})
	if typ != nvType {
		return nil
	}

	return func(dst, src reflect.Value) error {
		if !src.CanSet() {
			return nil
		}
		if !(src.Type() == nvType && dst.Type() == nvType) {
			return fmt.Errorf("expected to merge %s types, got %s and %s",
				nvType, src.Type(), dst.Type())
		}
		for i := 0; i < src.Len(); i++ {
			srcNV := src.Index(i)
			srcName := srcNV.FieldByName("Name")
			var overrideValue reflect.Value
			var dstValue reflect.Value
			for j := 0; j < dst.Len(); j++ {
				dstName := dst.Index(j).FieldByName("Name")
				if srcName.Equal(dstName) {
					dstValue = dst.Index(j).FieldByName("Value")
					overrideValue = srcNV.FieldByName("Value")
					break
				}
			}
			if overrideValue.IsValid() {
				dstValue.Set(overrideValue)
				continue
			}
			dst.Set(reflect.Append(dst, srcNV))
		}
		return nil
	}
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
	IMDS           *IMDSEnvSource           `json:"imds,omitempty"`
	S3             *S3EnvSource             `json:"s3,omitempty"`
	SecretsManager *SecretsManagerEnvSource `json:"secrets-manager,omitempty"`
	SSM            *SSMEnvSource            `json:"ssm,omitempty"`
}

func (e *EnvFrom) Validate() error {
	var (
		envNames []string
		errs     error
	)
	if e.IMDS != nil {
		if len(e.IMDS.Name) == 0 {
			err := fmt.Errorf("imds name is required")
			errs = errors.Join(errs, err)
		}
		envNames = append(envNames, "imds")
	}
	if e.S3 != nil {
		envNames = append(envNames, "s3-object")
	}
	if e.SecretsManager != nil {
		envNames = append(envNames, "secrets-manager")
	}
	if e.SSM != nil {
		envNames = append(envNames, "ssm")
	}
	lenEnvNames := len(envNames)
	if lenEnvNames > 1 {
		err := fmt.Errorf("expected 1 environment source, got %d: %s",
			lenEnvNames, strings.Join(envNames, ", "))
		errs = errors.Join(errs, err)
	}
	if errs != nil {
		return fmt.Errorf("env-from: %w", errs)
	}
	return nil
}

type IMDSEnvSource struct {
	Name     string `json:"name,omitempty"`
	Path     string `json:"path,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type S3EnvSource struct {
	Base64Encode bool   `json:"base64-encode,omitempty"`
	Bucket       string `json:"bucket,omitempty"`
	Key          string `json:"key,omitempty"`
	Name         string `json:"name,omitempty"`
	Optional     bool   `json:"optional,omitempty"`
}

type SecretsManagerEnvSource struct {
	Base64Encode bool   `json:"base64-encode,omitempty"`
	Name         string `json:"name,omitempty"`
	Optional     bool   `json:"optional,omitempty"`
	SecretID     string `json:"secret-id,omitempty"`
}

type SSMEnvSource struct {
	Base64Encode bool   `json:"base64-encode,omitempty"`
	Name         string `json:"name,omitempty"`
	Optional     bool   `json:"optional,omitempty"`
	Path         string `json:"path,omitempty"`
}

type Volume struct {
	EBS            *EBSVolumeSource            `json:"ebs,omitempty"`
	SecretsManager *SecretsManagerVolumeSource `json:"secrets-manager,omitempty"`
	SSM            *SSMVolumeSource            `json:"ssm,omitempty"`
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
	if v.SSM != nil {
		volumeNames = append(volumeNames, "ssm")
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
		// Only EBS volumes have actual mount points, so ignore the rest.
		if v.EBS != nil {
			mountPoints = append(mountPoints, v.EBS.Mount.Destination)
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
	Mount    Mount  `json:"mount,omitempty"`
	SecretID string `json:"secret-id,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type SSMVolumeSource struct {
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
	Destination string   `json:"destination,omitempty"`
	GroupID     *int     `json:"group-id,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	Options     []string `json:"options,omitempty"`
	UserID      *int     `json:"user-id,omitempty"`
}

type SecurityContext struct {
	ReadonlyRootFS bool `json:"readonly-root-fs,omitempty"`
	RunAsGroupID   *int `json:"run-as-group-id,omitempty"`
	RunAsUserID    *int `json:"run-as-user-id,omitempty"`
	SSHD           SSHD `json:"sshd,omitempty"`
}

type SSHD struct {
	Enable bool `json:"enable,omitempty"`
}

func ValidateServices(services []string) error {
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

func p[T any](v T) *T {
	return &v
}
