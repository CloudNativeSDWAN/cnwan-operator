# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.1] (2021-09-02)

### Added

- `/artifacts/secrets` folder to contain secrets (these are git ignored).
- `/artifacts/deploy` to contain yamls to deploy to the cluster.
- `/artifacts/settings` to contains settings for the operator and service
    registries.
- `/artifacts/deploy/other` to contain yaml files to deploy with the operator.

### Changed

- Fix an error causing `context.DeadlineExceeded` not being correctly
    parsed when calls to Service Directory fail.
- Update packages for Service Directory to the latest version.
- Update packages for etcd to a stable version.
- Files to deploy are now moved to `/artifacts`.
- `deploy.sh` is updated to reflect files reorganization.
- `remove.sh` is updated to reflect files reorganization.
- Update installation to include new ways to add files.
- Update go to `1.17`.

### Removed

- Some unused entrypoints in `Makefile`.
- Files that belonged to the old advance installation.
- `hack` folder.
- Documentation about the advance installation.

## [0.5.0] (2021-08-10)

### Added

- Package `cluster` now contains code to automatically pull some data from GKE
    in case it is running there.
- Package `cluster` now contains code to pull some resources from the cluster
    it is running in, e.g. secrets and configmaps.
- From previous point, it is able to automatically get region and project from
    GCP and automatically create the client with those data.


### Changed

- Settings for Google Service Directory can now be empty, and if so cloud
    metadata is used in case the cluster is running in GKE. It fails otherwise.
- Using `google.golang.org/genproto/googleapis/cloud/servicedirectory/v1` instead
    of `v1beta`
- Using `cloud.google.com/go/servicedirectory/apiv1` instead of `v1beta`
- The two points above required a change in some of the structures, such as
    changing `Metadata` with `Annotations` in services API.
- Service Directory handler can now be instantiated directly.
- Changed `project` to `ProjectID` in Service Directory handler.
- Changed `region` to `DefaultRegion` in Service Directory handler.
- Changed `--img` to `--image` in installation script.
- Dockerfile is updated by also including the new `utils.go`.
- The etcd credentials are now being retrieved automatically from the
    cluster.
- The Google service account is now retrieved automatically from within the
    cluster.
- Operator's settings configmap is now retrieved automatically from within the
    cluster.
    
### Removed

- Secrets and configmaps are not mounted on the pod anymore.
- Old code that was used to read the aforementioned files from the pod's
    mounted volumes.
- Old code from viper (will be removed entirely in future).


## [0.4.0] (2021-06-25)

### Added

- Package `cluster` which contains code that performs operations on the cluster
    that hosts the operator.
- Automatic cloud metadata pull from GCP and AWS (although the latter is not
    being fully used yet).
- Get network and subnetwork data from GCP and AWS.
- Automatically retrieve Google service account `Secret` from Kubernetes
    without mounting).
- `cloudMetadata` field in settings.
- Documentation on how to install `etcd` on the cluster.

### Changed

- Broker now has *persistent metadata* that are **always** inserted in services
    annotations/metadata on the service registry.
- `.gitignore` now also includes `*.bak*` files.
- Fixed some code typos such as `&*`.

## [0.3.0] (2021-01-22)

### Added

- `etcd` package that wraps around an etcd client
- `KeyBuilder` for easily building an etcd key
- A `Role` for reading secrets on the cluster
- A `RoleBinding` to bind the above role to Operator's service account
- `etcd` documentation on folder `docs/etcd`
- `service_registry.md` documentation about service registry and its objects
- `update.md` documentation
- `fakeKV` and `fakeTXN` to mock etcd key-value and transactions
- namespace name as environment variable
- `serviceRegistry` field in settings
- new utility functions in `utils`
- go report badge on readme.md

### Changed

- `Service directory` documentation is moved to its own folder on `docs/gcp_service_directory`
- main now uses `Goexit` for safer exit, but whole function will be changed in future
- different exit codes depending on the error
- service registry objects now contain struct tags
- new settings format which deprecates the old one
- `gcloud` in settings moved to `serviceRegistry.gcpServiceDirectory`
- `deploy.sh` and `remove.sh` adapted to work with etcd and work as flag-enabled CLIs
- git and docker badges changed with latest semver instead of latest date

## [0.2.1] (2020-10-19)

### Added

- A service account, so that the operator does not use the default one anymore
- Folder `deploy` containing pre-built yaml files, for an easier and
quicker deployment.
- Scripts `deploy.sh` and `remove.sh` to automate some commands.

### Changed

- RBAC is changed: role only asks for the bare minimum permissions it needs.
- Version format.

### Removed

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
