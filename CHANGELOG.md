# Changelog

## [0.6.0] - 2026-02-03

### Added

- Add container image source to ctr2image, instead of assuming a remote image. This will enable building AMIs from a local image in the future.
- Add a number of new tests.
- Add `SSL_CERT_FILE` parameter to the kernel command line, which will be passed to init as an environment variable.
- Add `version` subcommand to print the easyto version.
- Add a new *fast* build mode. When this is enabled, the Packer build instance will have easyto preinstalled instead of being pushed to the instance duing build. By default, an official AMI with easyto preinstalled will be searched for in the user's AWS account and in the easyto AWS account. If not found, easyto will fall back to slow mode. New command line options have also been added to enable specifying a custom build AMI.
- Add a Dockerfile for the official build image.
- Add a `copy-builder` subcommand to make it easy to copy the official build AMI to your own AWS account.
- Add the ability to make images public.

### Changed

- Update Go to 1.25.
- Modify functions to take an AferoFS parameter wherever possible to enable easier testing.
- Update documentation with fixes and new functionality.
- Update Actions workflow to publish official images.

### Removed

- Remove link from /proc/net/pnp to /etc/resolv.conf. Network configuration is now done by easyto-init.
- Clean up obsolete and unused code.
- Remove external dependencies during provisioning. The ctr2disk command now handles partitioning and creation of filesystems rather than using external utilities before running it. The unmounting of filesystems after provisioning is also now done in ctr2disk instead of by external utilities after it runs.
- Remove symlink from /.easyto/lib/modules to /lib/modules. The included modprobe command now has the /.easyto path compiled in.

## [0.5.0] - 2026-01-03

### Changed

- Update easyto-assets to v0.5.0. This updates the kernel to 6.12.63.
- Update easyto-init to v0.3.0. The new version includes modification of network configuration, bug fixes, and refactorings that do not change the interface.

## [0.4.0] - 2025-10-16

### Added

- Add CLI option to choose SSH interface for image builder.

### Changed

- Update easyto-init to v0.2.0. This enables attaching of EBS volumes at runtime based on tags.
- Update `github.com/docker/docker` and `github.com/ulikunitz/xz` dependencies for security advisories.

## [0.3.0] - 2024-11-10

### Changed

- Update easyto-assets to v0.4.0 to speed boot time.
- Update README to clarify behavior of `secrets-manager` volume.

### Removed

- Remove init from this repository. It has been replaced with a version developed in its [own repository](https://github.com/cloudboss/easyto-init).

## [0.2.0] - 2024-08-06

### Added

- Add a changelog, to follow the style of [keep a changelog](https://keepachangelog.com/en/1.0.0/).
- Add a `test` target to the Makefile that runs `go vet` and `go test`.
- Add GitHub workflows for test and release.
- Environment variables and command arguments in user data can include variable expansion in the style of Kubernetes [dependent environment variables](https://kubernetes.io/docs/tasks/inject-data-application/define-interdependent-environment-variables/).
- EC2 instance metadata can be used as a source for environment variables.
- Set a default `PATH` environment variable for the instance command on boot if it is not defined in the container image or user data.

### Changed

- Validate the VERSION variable in the Makefile.
- Pass --rm to docker run commands in the Makefile.
- Use the AWS SDK to retrieve EC2 instance metadata instead of using HTTP directly.
- Update instances to use cgroups v2.
- Update docker library.
- Tidy go.mod.
- Update easyto-assets to `v0.3.0` to include kmod for loading of kernel modules.

### Removed

- Building of assets has been moved to the `github.com/cloudboss/easyto-assets` repo.

## [0.1.0] - 2024-07-08

Initial release

[0.6.0]: https://github.com/cloudboss/easyto/releases/tag/v0.6.0
[0.5.0]: https://github.com/cloudboss/easyto/releases/tag/v0.5.0
[0.4.0]: https://github.com/cloudboss/easyto/releases/tag/v0.4.0
[0.3.0]: https://github.com/cloudboss/easyto/releases/tag/v0.3.0
[0.2.0]: https://github.com/cloudboss/easyto/releases/tag/v0.2.0
[0.1.0]: https://github.com/cloudboss/easyto/releases/tag/v0.1.0
