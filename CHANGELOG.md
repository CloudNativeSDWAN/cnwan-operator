# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1] (2020-10-19)

## Added

- A service account, so that the operator does not use the default one anymore
- Folder `deploy` containing pre-built yaml files, for an easier and
quicker deployment.
- Scripts `deploy.sh` and `remove.sh` to automate some commands.

## Changed

- RBAC is changed: role only asks for the bare minimum permissions it needs.
- Version format.

## Removed

- Annotations list in `config/manager/settings.yaml` is now empty.
- Leader election and metrics server
- Many resources that are not utilized.

## [0.2.0] (2020-09-24)

### Added

- New *Service Registry Broker*, which manages data - i.e. checks if data is
correct or if already exists, etc. - before sending requests to the service
registry. As a matter of fact, it performs operations on namespaces, services
and endpoints before actually executing the appropriate functions of the
service registry. The service registry library can be used, but letting
everything go through the broker is recommended as it will set up the data
in the correct way and format.
- Stronger unit tests for the service registry broker.
- New handler for Google Cloud Service Directory, with better testing.
- New "intermediate" types: the operator works with `Namespace`, `Service`
and `Endpoint` types, which strip away the complexities and non-relevant
data from the K8s types or the ones used by the service registry.
- Timeouts: all HTTP/S requests made by the operator to the service registry
are now subject to a timeout. If the timeout expires, the http call is
interrupted. This avoids the operator being stuck on requests and accumulate
too many resources.
- This Changelog.
- Functions have more logs.
- Readme: add Kubernetes version requirement.
- Readme: add `Ownership` section.
- Readme: add `Kubernetes Requirements` section.

### Changed

- Code about service registry is moved to `/pkg`.
- `types` and `utils` are now moved to `/internal`.
- `Dockerfile` has been changed accordingly
- Improve requirements by adding minimum version to some of the dependencies
of the operator.
- Upgrades:
  - `sigs.k8s.io/controller-runtime` to `v0.6.3`
  - `google.golang.org/grpc` to `v1.33.0`
  - `github.com/stretchr/testify` to `v1.6.1`
  - `github.com/spf13/viper` to `v1.7.1`
  - `github.com/onsi/gomega` to `v1.10.3`
  - `github.com/onsi/ginkgo` to `v1.14.2`
  - `github.com/googleapis/gax-go` to `v1.0.3`
  - `cloud.google.com/go` to `v0.69.1`

### Fixed

- Readme: fixed a typo in `Service Directory Settings` anchor in table of contents.

### Removed

- The old `servicedirectory` package was removed, in favor of
`pkg/servregistry/gcloud/servicedirectory` containing better isolation,
separation of concerns and unit tests.
- `utils` has been cleaned up to only contain `FilterAnnotations`, as the
other functions have now been moved to other packages or just not used anymore.
- `COPYRIGHT` file is removed, as copyright is contained on top of each file
created by the CN-WAN Operator Owners.

## [0.1.0] (2020-08-12)

### Added

- Namespace and Service controllers are added.
- Internal structures such as `types` and `utils`.
- Support for Google Cloud Service Directory.
