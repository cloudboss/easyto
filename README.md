# easyto

If you have a container image that you want to run directly on EC2, then easyto is for you. It builds an EC2 AMI from a container image.

## How does it work?

It creates a temporary EC2 AMI build instance[^1] with an EBS volume attached. The EBS volume is partitioned and formatted, and the container image layers are written to the main partition. Then a Linux kernel, bootloader, and custom init and utilities are added. The EBS volume is then snapshotted and an AMI is created from it.

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

### Command line options

The `ami` subcommand takes the following options:

`--ami-name` or `-a`: (Required) -  Name of the AMI, which must follow the name [constraints](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RegisterImage.html) defined by Amazon.

`--container-image` or `-c`: (Required) - Name of the container image.

`--subnet-id` or `-s`: (Required) - ID of the subnet in which to run the image builder.

`--services`: (Optional, default `chrony`) - Comma separated list of services to enable, which may include `chrony`, `ssh`. Use an empty string to disable all services.

`--size` or `-S`: (Optional, default `10`) - Size of the image root volume in GB.

`--login-shell`: (Optional, default `/.easyto/bin/sh`) - Shell to use for the login user if ssh service is enabled.

`--login-user`: (Optional, default `cloudboss`) - Login user to create in the VM image if ssh service is enabled.

`--asset-directory` or `-A`: (Optional) - Path to a directory containing asset files. Normally not needed unless changing the layout of directories contained in the release.

`--packer-directory` or `-P` (Optional) - Path to a directory containing packer and its configuration. Normally not needed unless changing the layout of directories contained in the release.

`--root-device-name`: (Optional, default `/dev/xvda`) - Name of the AMI root device.

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
      fs-type: ext4
      make-fs: true
      mount:
        destination: /var/lib/postgresql
```

See the [examples](./examples) folder for more.

The full specification is as follows:

`args`: (Optional, type _list of string_, default is dependent on the image and the value of `command`) - Arguments to `command`. If `args` is not defined in user data, it defaults to the container image [cmd](https://docs.docker.com/reference/dockerfile/#cmd), unless `command` is defined in user data, in which case it defaults to an empty list.

`command`: (Optional, type _list of string_, default is the image [entrypoint](https://docs.docker.com/reference/dockerfile/#entrypoint), if defined) - Override of the image's entrypoint.

`debug`: (Optional, type _bool_, default `false`) - Whether or not to enable debug logging.

`disable-services`: (Optional, type _list of _string_, default `[]`) - A list of services to disable at runtime if they were included in the image, e.g. with `easyto ami --services=[...]`.

`env`: (Optional, type _list_ of [_name-value_](#name-value-object) objects, default `[]`) - The names and values of environment variables to be passed to `command`.

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

`s3`: (Optional, type [_s3-env_](#s3-env-object) object) - Configuration for an S3 environment source.

`ssm`: (Optional, type [_ssm-env_](#ssm-env-object) object) - Configuration for an SSM environment source.

`secrets-manager`: (Optional, type [_secrets-manager-env_](#secrets-manager-env-object) object) - Configuration for a Secrets Manager environment source.

#### s3-env object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `s3:GetObject` for the object at `bucket` and `key`.

`base64-encode`: (Optional, type _bool_, default `false`) - Whether or not to Base64 encode the value, if `name` is defined.

`bucket`: (Required, type _string_) - The name of the S3 bucket.

`key`: (Required, type _string_) - The name of the object in the S3 bucket.

`name`: (Optional, type _string_) - If defined, this will be the name of the environment variable and the S3 object's value will be the value. If not defined, the object's value must contain JSON key/value strings one level deep, whose keys will be the environment variable names and whose values will be the values.

> [!WARNING]
> There is no check on the size of the value; values too large can prevent `command` from starting.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the object is optional. If `true`, then a failure to fetch the object will not be treated as an error.

#### ssm-env object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `ssm:GetParameter`, `ssm:GetParametersByPath`, and `kms:Decrypt` for the KMS key used to encrypt the parameter if it is of type `SecureString` and a customer-managed key was used.

`base64-encode`: (Optional, type _bool_, default `false`) - Whether or not to Base64 encode the value, if `name` is defined.

`name`: (Optional, type _string_) - If defined, this will be the name of the environment variable and the SSM parameter's value will be the value. If not defined, the parameter's value must contain JSON key/value strings one level deep, whose keys will be the environment variable names and whose values will be the values.

> [!WARNING]
> There is no check on the size of the value; values too large can prevent `command` from starting.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the SSM parameter is optional. If `true`, then a failure to fetch the parameter will not be treated as an error.

`path`: (Required, type _string_) - The SSM Parameter path, which must resolve to a single parameter.

#### secrets-manager-env object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `secretsmanager:GetSecretValue`, and `kms:Decrypt` for the KMS key used to encrypt the secret if a customer-managed key was used.

`base64-encode`: (Optional, type _bool_, default `false`) - Whether or not to Base64 encode the value, if `name` is defined.

`name`: (Optional, type _string_) - If defined, this will be the name of the environment variable and the secret's value will be the value. If not defined, the secret's value must contain JSON key/value strings one level deep, whose keys will be the environment variable names and whose values will be the values.

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

`device`: (Required, type _string_) - Name of the device as defined in the EC2 instance's block device mapping.

`fs-type`: (Required, type _string_) - Filesystem type of the device. Available types are `ext2`, `ext3`, `ext4`, and `btrfs`. The filesystem will be formatted on the first boot.

`mount`: (Required, type [_mount_](#mount-object) object) - Configuration of the mount for the EBS volume.

#### s3-volume object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `s3:GetObject` and  `s3:ListObjects`.

An S3 volume is a pseudo-volume, as the parameters from S3 are copied as files to the object's `mount.destination` one time on boot. The owner and group of the files defaults to `security.run-as-user` and `security.run-as-group` unless explicitly specified in the volume's `mount.user-id` and `mount.group-id`.

`bucket`: (Required, type _string_) - Name of the S3 bucket.

`key-prefix`: (Optional, type _string_, default is blank) - Only objects in `bucket` beginning with this prefix will be returned. If not defined, the whole bucket will be copied.

> [!WARNING]
> If `key-prefix` contains S3 objects such as `abc/xyz` and also `abc/xyz/123`, you may get an error `not a directory` on boot because there cannot be both a file and directory with the same name.

`mount`: (Required, type [_mount_](#mount-object) object) - Configuration of the destination for the S3 objects.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the S3 objects are optional. If `true`, then a failure to fetch the objects will not be treated as an error.

#### ssm-volume object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `ssm:GetParameter`, `ssm:GetParametersByPath`, and `kms:Decrypt` for the KMS key used to encrypt the parameter if they are of type `SecureString` and a customer-managed key was used.

An SSM volume is a pseudo-volume, as the parameters from SSM Parameter Store are copied as files to the object's `mount.destination` one time on boot. Any updates to the parameters would require a reboot to get the new values. The files are always written with permissions of `0600`, even if the parameters are not of type `SecureString`. The owner and group of the files defaults to `security.run-as-user` and `security.run-as-group` unless explicitly specified in the volume's `mount.user-id` and `mount.group-id`.

`path`: (Required, type _string_) - The SSM parameter path. If the path begins with `/` and has parameters below it, everything under it will be retrieved and stored in files named the same as the parameters under `mount.destination`, omitting the leading `path`. The SSM parameters can be nested, and those with child parameters will be used to create subdirectories below them. If `path` is the full path of a single parameter or does not begin with `/`, it must resolve to a single parameter, and `mount.destination` will be the file name on disk.

> [!WARNING]
> If `path` contains SSM parameters such as `abc/xyz` and also `abc/xyz/123`, you may get an error `not a directory` on boot because there cannot be both a file and directory with the same name.

`mount`: (Required, type [_mount_](#mount-object) object) - Configuration of the destination for the SSM parameters.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the parameters are optional. If `true`, then a failure to fetch the parameters will not be treated as an error.

#### secrets-manager-volume object

> [!NOTE]
> The EC2 instance must have an instance profile with permission to call `secretsmanager:GetSecretValue`, and `kms:Decrypt` for the KMS key used to encrypt the secret if a customer-managed key was used.

A Secrets Manager volume is a pseudo-volume, as the secret from Secrets Manager is copied as a file to the path defined in `mount.destination` one time on boot. Any updates to the secret would require a reboot to get the new value. The file is always written with a mode of `0600`. The owner and group of the file defaults to `security.run-as-user` and `security.run-as-group` unless explicitly specified in the volume's `mount.user-id` and `mount.group-id`.

`mount`: (Required, type [_mount_](#mount-object) object) - Configuration of the destination for the secret.

`optional`: (Optional, type _bool_, default `false`) - Whether or not the secret is optional. If `true`, then a failure to fetch the secret will not be treated as an error.

`secret-id`: (Required, type _string_) - The name or ARN of the secret. If it is in another AWS account, the ARN must be used.

#### mount object

`destination`: (Required, type _string_) - The mount destination. This may be a file or a directory depending on the configuration of the volume.

`group-id`: (Optional, type _int_, default `0`) - The group ID of the destination.

`mode`: (Optional, type _string_, default `0755`) - The mode of the destination.

`options`: (Optional, type _list_ of _string_, default `[]`) - Options for filesystem mounting, dependent on the filesystem type. These are the options that would be passed to the `mount` command with `-o`.

`user-id`: (Optional, type _int_, default `0`) - The user ID of the destination.

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
