# easyto

If you have a container image that you want to run directly on EC2, then easyto is for you. It builds an EC2 AMI from a container image.

## How does it work?

It creates a temporary EC2 AMI build instance[^1] with an EBS volume attached. The EBS volume is partitioned and formatted, and the container image layers are written to the main partition. Then a Linux kernel, bootloader, and [custom init](https://github.com/cloudboss/easyto-init) and utilities are added. The EBS volume is then snapshotted and an AMI is created from it.

The `metadata.json` from the container image is written into the AMI so init will know what command to start on boot, and behave as specified in the Dockerfile. The command can be overridden, much like you can with docker or Kubernetes. This is accomplished with a custom [EC2 user data](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-add-user-data.html) format [defined below](#user-data) that is intended to be similar to a Kubernetes pod definition.

[![Screencast](https://img.youtube.com/vi/lruK2WOWa-o/0.jpg)](https://www.youtube.com/watch?v=lruK2WOWa-o)

## Installing

Download the release and unpack it. The `easyto` binary lives in the `bin` subdirectory and runs directly from there.

> [!NOTE]
> Please read through the [command line options](#command-line-options) if you decide to change the layout of directories contained in the release archive.

## Building an image

First make sure your AWS credentials and region are in scope, for example:

```
export AWS_PROFILE=xyz
export AWS_REGION=us-east-1
```

To create an AMI, run the `ami` subcommand. For example, to create an AMI called `postgres-16.2-bullseye` from the `postgres:16.2-bullseye` container image that includes both chrony and ssh services, run:

```
easyto ami -a postgres-16.2-bullseye -c postgres:16.2-bullseye -s subnet-e358acdfe25b8fb3b --services=chrony,ssh
```

### Fast vs slow mode

The `ami` subcommand runs in one of *fast* or *slow* build modes.

In fast mode, the build instance is created from an AMI that has easyto preinstalled. This is the default when the builder AMI matching your easyto version is available in your region. The AMI with the name `ghcr.io-cloudboss-easyto-builder-<version>` is searched for in your own account and in the official easyto account, where `<version>` is the same as the output of running `easyto version`.

In slow mode, the build uses a Debian base AMI and copies easyto to the instance during the build process. This is slower but works without needing the builder AMI to be available. Slow mode is used automatically as a fallback when the fast mode AMI is not found.

You can also specify a custom builder image with `--builder-image`, with `--builder-image-mode` to specify fast mode if easyto is preinstalled on it. When easyto is preinstalled, it must be the full bundle including assets directory, and the `easyto` executable or a link to it must be on the `PATH`.

### Command line options

The `ami` subcommand takes the following options:

`--ami-name` or `-a`: (Required) -  Name of the AMI, which must follow the name [constraints](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RegisterImage.html) defined by Amazon.

`--container-image` or `-c`: (Required) - Name of the container image from which the AMI is derived.

`--subnet-id` or `-s`: (Required) - ID of the subnet in which to run the image builder.

`--services`: (Optional, default `chrony`) - Comma separated list of services to enable, which may include `chrony`, `ssh`. Use an empty string to disable all services.

`--size` or `-S`: (Optional, default `10`) - Size of the image root volume in GB.

`--login-shell`: (Optional, default `/.easyto/bin/sh`) - Shell to use for the login user if ssh service is enabled.

`--login-user`: (Optional, default `cloudboss`) - Login user to create in the AMI if ssh service is enabled.

`--builder-image`: (Optional) - AMI name pattern or ID for the builder image. If not specified, uses the easyto builder AMI matching the current version, falling back to Debian if not found.

`--builder-image-login-user`: (Optional, default `cloudboss`) - SSH login user for the builder image when using `--builder-image`.

`--builder-image-mode`: (Optional, default `slow`) - Build mode to use with `--builder-image` and has no effect if it is not defined. Must be one of `fast` or `slow`.

`--asset-directory` or `-A`: (Optional) - Path to a directory containing asset files. Normally not needed unless changing the layout of directories contained in the release.

`--packer-directory` or `-P` (Optional) - Path to a directory containing packer and its configuration. Normally not needed unless changing the layout of directories contained in the release.

`--root-device-name`: (Optional, default `/dev/xvda`) - Name of the AMI root device.

`--ssh-interface`: (Optional, default `public_ip`) - The SSH interface to use to connect to the image builder. This must be one of `public_ip` or `private_ip`.

`--debug`: (Optional) - Enable debug output.

`--help` or `-h`: (Optional) - Show help output.

## Running an instance

Instances are created "the usual way" with the AWS console, AWS CLI, or Terraform, for example. Modifying the startup configuration is different from other EC2 instances however, because the [user data](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-add-user-data.html) format is different. The AMIs are not configured to use [cloud-init](https://cloudinit.readthedocs.io/en/latest/index.html), so just putting a shell script into user data will not work.

### User data

The user data format is meant to be similar to a container configuration, and borrows some of its nomenclature from the Kubernetes pod spec.

Example:

```
env-from:
  - ssm:
      path: /database/abc/credentials
volumes:
  - ebs:
      device: /dev/sdb
      mount:
        destination: /var/lib/postgresql
        fs-type: ext4
```

See the [examples](./examples) folder for more.

The full specification is as follows:

`args`: (Optional, type _list_ of _string_, default is dependent on the image and the value of `command`) - Arguments to `command`. If `args` is not defined in user data, it defaults to the container image [cmd](https://docs.docker.com/reference/dockerfile/#cmd), unless `command` is defined in user data, in which case it defaults to an empty list. Elements in the list can refer to variables defined in `env` and `env-from` (see [variable expansion](#variable-expansion)).

`command`: (Optional, type _list_ of _string_, default is the image [entrypoint](https://docs.docker.com/reference/dockerfile/#entrypoint), if defined) - Override of the image's entrypoint. Elements in the list can refer to variables defined in `env` and `env-from` (see [variable expansion](#variable-expansion)).

`debug`: (Optional, type _bool_, default `false`) - Whether or not to enable debug logging.

`disable-services`: (Optional, type _list_ of _string_, default `[]`) - A list of services to disable at runtime if they were included in the image, e.g. with `easyto ami --services=[...]`.

`env`: (Optional, type _list_ of [_name-value_](#name-value-object) objects, default `[]`) - The names and values of environment variables to be passed to `command`. Values can refer to variables defined in `env` and `env-from` (see [variable expansion](#variable-expansion)).

> [!NOTE]
> If the `PATH` environment variable is not set in the container image or user data, a default `PATH` of `/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin` will be set on boot.

`env-from`: (Optional, type _list_ of [_env-from_](#env-from-object) objects, default `[]`) - Environment variables to be passed to `command`, to be retrieved from the given sources.

`init-scripts`: (Optional, type _list_ of _string_, default `[]`) - A list of scripts to run on boot. They must start with `#!` and have a valid interpreter available in the image. For lightweight images that have no shell in the container image they are derived from, `/.easyto/bin/busybox sh` can be used. The AMI will always have `/.easyto/bin/busybox` available as a source of utilities that can be used in the scripts. Init scripts run just before any services have started and `command` is executed.

> Example:
>
> ```
> init-scripts:
>   - |
>     #!/.easyto/bin/busybox sh
>     bb=/.easyto/bin/busybox
>     (umask 277; ${bb} cp /path/to/secret /some/other/path)
> ```

`replace-init`: (Optional, type _bool_, default `false`) - If `true`, `command` will replace init when executed. This may be useful if you want to run your own init process. However, easyto init will still do everything leading up to the execution of `command`, for example formatting and mounting filesystems defined in `volumes` and setting environment variables.

`security`: (Optional, type [_security_](#security-object) object, default `{}`) - Configuration of security settings.

`shutdown-grace-period`: (Optional, type _int_, default `10`) - When shutting down, the number of seconds to wait for all processes to exit cleanly before sending a kill signal.

`sysctls`: (Optional, type _list_ of [_name-value_](#name-value-object) objects, default `[]`) - The names and values of sysctls to set before starting `command`.

`volumes`: (Optional, type _list_ of [_volume_](#volume-object) objects, default `[]`) - Configuration of volumes.

`working-dir`: (Optional, type _string_, default is dependent on the container image) - The directory in which `command` will be run. This defaults to the container image's [workdir](https://docs.docker.com/reference/dockerfile/#workdir) if it is defined, or else `/`.

#### name-value object

`name`: (Required, type _string_) - Name of item.

`value`: (Required, type _string_) - Value of item.

#### env-from object

The following sources are available for environment variables. Each can be specified multiple times.

`imds`: (Optional, type [_imds-env_](#imds-env-object) object) - Configuration for an [IMDS](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html) environment source.

`s3`: (Optional, type [_s3-env_](#s3-env-object) object) - Configuration for an S3 environment source.

`ssm`: (Optional, type [_ssm-env_](#ssm-env-object) object) - Configuration for an SSM environment source.

`secrets-manager`: (Optional, type [_secrets-manager-env_](#secrets-manager-env-object) object) - Configuration for a Secrets Manager environment source.

#### imds-env object

`name`: (Required, type _string_) - The name of the environment variable.

`path`: (Required, type _string_) - The path of the metadata value. Paths start after `/latest/meta-data`. A leading slash is not required. For example, to get the AWS region, use `placement/region`.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the variable is optional. If `true`, then a failure to fetch the value will not be treated as an error.

#### s3-env object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `s3:GetObject` for the object at `bucket` and `key`.

`base64-encode`: (Optional, type _bool_, default `false`) - Whether or not to Base64 encode the value, if `name` is defined.

`bucket`: (Required, type _string_) - The name of the S3 bucket.

`key`: (Required, type _string_) - The name of the object in the S3 bucket.

`name`: (Optional, type _string_) - If defined, this will be the name of the environment variable and the S3 object's value will be the value. If not defined, the object's contents must contain JSON key/value strings one level deep, whose keys will be the environment variable names and whose values will be the values.

> [!WARNING]
> There is no check on the size of the value; values too large can prevent `command` from starting.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the object is optional. If `true`, then a failure to fetch the object will not be treated as an error.

#### ssm-env object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `ssm:GetParameter`, `ssm:GetParametersByPath`, and `kms:Decrypt` for the KMS key used to encrypt the parameter if it is of type `SecureString` and a customer-managed key was used.

`base64-encode`: (Optional, type _bool_, default `false`) - Whether or not to Base64 encode the value, if `name` is defined.

`name`: (Optional, type _string_) - If defined, this will be the name of the environment variable and the SSM parameter's value will be the value. If not defined, the parameter's contents must contain JSON key/value strings one level deep, whose keys will be the environment variable names and whose values will be the values.

> [!WARNING]
> There is no check on the size of the value; values too large can prevent `command` from starting.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the SSM parameter is optional. If `true`, then a failure to fetch the parameter will not be treated as an error.

`path`: (Required, type _string_) - The SSM Parameter path, which must resolve to a single parameter.

#### secrets-manager-env object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `secretsmanager:GetSecretValue`, and `kms:Decrypt` for the KMS key used to encrypt the secret if a customer-managed key was used.

`base64-encode`: (Optional, type _bool_, default `false`) - Whether or not to Base64 encode the value, if `name` is defined.

`name`: (Optional, type _string_) - If defined, this will be the name of the environment variable and the secret's value will be the value. If not defined, the secret's contents must contain JSON key/value strings one level deep, whose keys will be the environment variable names and whose values will be the values.

> [!WARNING]
> There is no check on the size of the value; values too large can prevent `command` from starting.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the secret is optional. If `true`, then a failure to fetch the secret will not be treated as an error.

`secret-id`: (Required, type _string_) - The name or ARN of the secret. If it is in another AWS account, the ARN must be used.

#### security object

`readonly-root-fs`: (Optional, type _bool_, default `false`) - Whether or not to mount the root filesystem as readonly. This happens after any services have initialized, just before `command` is executed. If `init-scripts` are defined, they will run before this.

`run-as-group-id`: (Optional, type _int_, default is dependent on the container image) - Group ID that `command` should run as. This defaults to the optional group ID from the container image's [user](https://docs.docker.com/reference/dockerfile/#user) if it is defined, or else `0`.

`run-as-user-id`: (Optional, type _int_, default is dependent on the container image) - User ID that `command` should run as. This defaults to the container image's [user](https://docs.docker.com/reference/dockerfile/#user) if it is defined, or else `0`.

#### volume object

`ebs`: (Optional, type [_ebs-volume_](#ebs-volume-object) object, default `{}`) - Configuration of an EBS volume.

`s3`: (Optional, type [_s3-volume_](#s3-volume-object) object, default `{}`) - Configuration of an S3 pseudo-volume.

`ssm`: (Optional, type [_ssm-volume_](#ssm-volume-object) object, default `{}`) - Configuration of an SSM Parameter pseudo-volume.

`secrets-manager`: (Optional, type [_secrets-manager-volume_](#secrets-manager-volume-object) object) - Configuration for Secrets Manager pseudo-volume.

#### ebs-volume object

`attachment`: (Optional, type _list_ of [_ebs-attachment_](#ebs-attachment-object)) - Configuration of EBS volume attachment, which enables a volume to be attached at runtime based on its tags.

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `ec2:AttachVolume` and `ec2:DescribeVolumes`.

`device`: (Required, type _string_) - Name of the device as defined in the EC2 instance's block device mapping.

`mount`: (Optional, type [_mount_](#mount-object) object) - Configuration of the mount for the EBS volume. If not defined, no filesystem will be formatted and the volume will not be mounted.

#### s3-volume object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `s3:GetObject` and  `s3:ListObjects`.

An S3 volume is a pseudo-volume, as the parameters from S3 are copied as files to the object's `mount.destination` one time on boot. The owner and group of the files defaults to `security.run-as-user-id` and `security.run-as-group-id` unless explicitly specified in the volume's `mount.user-id` and `mount.group-id`.

`bucket`: (Required, type _string_) - Name of the S3 bucket.

`key-prefix`: (Optional, type _string_, default is blank) - Only objects in `bucket` beginning with this prefix will be returned. If not defined, the whole bucket will be copied.

> [!WARNING]
> If `key-prefix` contains S3 objects such as `abc/xyz` and also `abc/xyz/123`, you may get an error `not a directory` on boot because there cannot be both a file and directory with the same name.

`mount`: (Required, type [_mount_](#mount-object) object) - Configuration of the destination for the S3 objects.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the S3 objects are optional. If `true`, then a failure to fetch the objects will not be treated as an error.

#### ssm-volume object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `ssm:GetParameter`, `ssm:GetParametersByPath`, and `kms:Decrypt` for the KMS key used to encrypt the parameter if they are of type `SecureString` and a customer-managed key was used.

An SSM volume is a pseudo-volume, as the parameters from SSM Parameter Store are copied as files to the object's `mount.destination` one time on boot. Any updates to the parameters would require a reboot to get the new values. The files are always written with permissions of `0600`, even if the parameters are not of type `SecureString`. The owner and group of the files defaults to `security.run-as-user-id` and `security.run-as-group-id` unless explicitly specified in the volume's `mount.user-id` and `mount.group-id`.

`path`: (Required, type _string_) - The SSM parameter path. If the path begins with `/` and has parameters below it, everything under it will be retrieved and stored in files named the same as the parameters under `mount.destination`, omitting the leading `path`. The SSM parameters can be nested, and those with child parameters will be used to create subdirectories below them. If `path` is the full path of a single parameter or does not begin with `/`, it must resolve to a single parameter, and `mount.destination` will be the file name on disk.

> [!WARNING]
> If `path` contains SSM parameters such as `abc/xyz` and also `abc/xyz/123`, you may get an error `not a directory` on boot because there cannot be both a file and directory with the same name.

`mount`: (Required, type [_mount_](#mount-object) object) - Configuration of the destination for the SSM parameters.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the parameters are optional. If `true`, then a failure to fetch the parameters will not be treated as an error.

#### secrets-manager-volume object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `secretsmanager:GetSecretValue`, and `kms:Decrypt` for the KMS key used to encrypt the secret if a customer-managed key was used.

A Secrets Manager volume is a pseudo-volume, as the secret from Secrets Manager is copied as a file to the path defined in `mount.destination` one time on boot. Any updates to the secret would require a reboot to get the new value. This volume results in a single file being written, not a directory tree as is possible with S3 and SSM volumes. The file is always written with a mode of `0600`. The owner and group of the file defaults to `security.run-as-user-id` and `security.run-as-group-id` unless explicitly specified in the volume's `mount.user-id` and `mount.group-id`.

`mount`: (Required, type [_mount_](#mount-object) object) - Configuration of the destination for the secret.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the secret is optional. If `true`, then a failure to fetch the secret will not be treated as an error.

`secret-id`: (Required, type _string_) - The name or ARN of the secret. If it is in another AWS account, the ARN must be used.

#### ebs-attachment object

`tags`: (Required, type [_tag-key-value_](#tag-key-value-object)) - A list of tags used to filter the EBS volume when calling `ec2:DescribeVolumes`.

`timeout`: (Optional, type _int_, default `300`) - How long to wait in seconds for the EBS volume to be available.

#### tag-key-value object

`key`: (Required, type _string_) - The name of the tag.

`value`: (Optional, type _string_) - The value of the tag. If not defined, only the key is used as a filter.

#### mount object

`destination`: (Required, type _string_) - The mount destination. This may be a file or a directory depending on the configuration of the volume.

`fs-type`: (Conditional, type _string_) - Filesystem type of the device. Available types are `ext2`, `ext3`, `ext4`, and `btrfs`. The filesystem will be formatted on the first boot. Required for EBS volumes and ignored otherwise.

`group-id`: (Optional, type _int_, default is the value of `security.run-as-group-id`) - The group ID of the destination.

`mode`: (Optional, type _string_, default `0755`) - The mode of the destination.

`options`: (Optional, type _list_ of _string_, default `[]`) - Options for filesystem mounting, dependent on the filesystem type. These are the options that would be passed to the `mount` command with `-o`.

`user-id`: (Optional, type _int_, default is the value of `security.run-as-user-id`) - The user ID of the destination.

#### Variable expansion

In `env`, `command`, and `args`, environment variables can be referenced by using the syntax `$(VAR)`.

To escape variables and prevent expansion, use double `$` characters; `$$(VAR)` will be passed as `$(VAR)`.

Environment variables defined in `env` can only reference other variables in `env` that are defined before them, whereas variables defined in `env-from` can be referenced anywhere in `env`. In `command` and `args`, any variable defined in `env` or `env-from` can be referenced.

Variables do not expand recursively. References to variables containing other variables will expand to their initial value, not the value after expansion.

Variables that are not found will be passed as-is with no attempt to expand them.

Example with a command:

```
command:
  - /application
  - --bind-address
  - $(IPV4_ADDRESS):8080
env-from:
  - imds:
      name: IPV4_ADDRESS
      path: local-ipv4
```

Example with multiple references:

```
env:
  - name: ABC
    value: 123
  - name: DEF
    value: 456
  - name: GHI
    value: $(ABC) and $(DEF)
```

Example with an environment variable referencing another resolved with `env-from`:

```
env:
  - name: PGPASSWORD
    # The value of `password` is resolved from Secrets Manager.
    value: $(password)
env-from:
  - secrets-manager:
      # The contents of /database/abc/credentials is a JSON mapping one level deep containing the key `password`.
      secret-id: /database/abc/credentials
```

## System services

AMIs can be configured at build time to run additional services on boot. The services available are [chrony](https://chrony-project.org/) and [ssh](https://www.openssh.com/).

To disable all services at build time, use `--services=""` when running `easyto ami`.

Even if the AMI is built with services, they can be disabled at runtime with `disable-services` in user data.

### Chrony

Chrony is included in AMIs by default and is configured to use Amazon's NTP server.

### SSH

The ssh server is not included in AMIs by default, but can be added with `--services=chrony,ssh` (or just `--services=ssh`) when running `easyto ami`.

If an ssh [key pair](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html) is not specified when creating an EC2 instance from the AMI, the ssh server will not start, even if it is enabled in the AMI.

The login user for ssh defaults to `cloudboss` with a shell of `/.easyto/bin/sh`, but these can be changed with the `--login-user` and `--login-shell` options to `easyto ami`.

## Shutdown behavior

The AMIs are configured to behave similarly to containers on shutdown. If the instance's command shuts down for any reason, the instance will shut down the same as if the EC2 API were called to stop the instance. All child processes and services will stop, filesystems will be unmounted, and the instance will power off. Termination of the instance must however be done with a target group health check or some other process.

## Limitations

* AMIs will be configured with UEFI boot mode, so only instance types that support UEFI boot can be used with them.

* The included utilities are intended to be just enough to bootstrap the image's command, and to provide a bare-bones environment for SSH logins with busybox.

* Only the amd64 architecture is currently supported.

## Roadmap

* Support arm64 architecture.

* Additional subcommands.
  * Validate user data.
  * Quick test of an image.

* Support instance store volumes.

[^1]: Packer is used with the [Amazon EBS Surrogate builder](https://developer.hashicorp.com/packer/integrations/hashicorp/amazon/latest/components/builder/ebssurrogate) to orchestrate the AMI build.
