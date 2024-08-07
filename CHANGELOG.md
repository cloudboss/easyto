# Changelog

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

[0.2.0]: https://github.com/cloudboss/easyto/releases/tag/v0.2.0
[0.1.0]: https://github.com/cloudboss/easyto/releases/tag/v0.1.0
