# CN-WAN Operator

![GitHub](https://img.shields.io/github/license/CloudNativeSDWAN/cnwan-operator)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/CloudNativeSDWAN/cnwan-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/CloudNativeSDWAN/cnwan-operator)](https://goreportcard.com/report/github.com/CloudNativeSDWAN/cnwan-operator)
![Kubernetes version](https://img.shields.io/badge/kubernetes-1.11.3%2B-blue)
![GitHub Workflow Status](https://img.shields.io/github/workflow/status/CloudNativeSDWAN/cnwan-operator/Test)
![GitHub release (latest SemVer including pre-releases)](https://img.shields.io/github/v/release/CloudNativeSDWAN/cnwan-operator?include_prereleases)
![Docker Image Version (latest SemVer)](https://img.shields.io/docker/v/cnwan/cnwan-operator?label=docker%20image%20version)
[![DevNet published](https://static.production.devnetcloud.com/codeexchange/assets/images/devnet-published.svg)](https://developer.cisco.com/codeexchange/github/repo/CloudNativeSDWAN/cnwan-operator)

Register and manage your Kubernetes Services to a Service Registry.

The CN-WAN Operator is part of the Cloud Native SD-WAN (CN-WAN) project. Please check the [CN-WAN documentation](https://github.com/CloudNativeSDWAN/cnwan-docs) for the general project overview and architecture. You can contact the CN-WAN team at [cnwan@cisco.com](mailto:cnwan@cisco.com).

## Overview

CN-WAN Operator is a Kubernetes operator created with [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) that watches for changes in services deployed in your cluster and registers them to a service registry so that clients can later discover all registered services and know how to connect to them properly.

The [service registry](./docs/service_registry.md) is populated with only the allowed resources and properties, as specified from a configuration file.

## Supported Service Registries

Currently, the CN-WAN Operator can use the popular key-value storage [etcd](https://etcd.io/) as a service registry or use other commercial solutions like Google Cloud's [Service Directory](https://cloud.google.com/service-directory).

We have a document about this topic so that you can learn more about it and we recommend you to read: [Service registry](./docs/service_registry.md).

If you're undecided or don't know which one to use or just want to try the project, you can try with [etcd](./docs/etcd/concepts.md).

## Try It Out

If you want to quickly see how CN-WAN Operator works, you can follow this simple step by step [quickstart with etcd](./docs/etcd/quickstart.md) guide.

## Documentation

* [Concepts](./docs/concepts.md)
* [Basic Installation](./docs/basic_installation.md)
* [Advanced Installation](./docs/advanced_installation.md)
* [Update](./docs/update.md)
* [Configuration](./docs/configuration.md)
* [Service registry](./docs/service_registry.md)

### etcd

* [Quickstart with etcd](./docs/etcd/quickstart.md)
* [Concepts](./docs/etcd/concepts.md)
* [Cluster setup](./docs/etcd/demo_cluster_setup.md)
* [Example interactions with etcd](./docs/etcd/interact.md)
* [Configure CN-Operator with etcd](./docs/etcd/operator_configuration.md)

### Google Service Directory

* [Quickstart with service directory](./docs/gcp_service_directory/quickstart.md)
* [Concepts](./docs/gcp_service_directory/concepts.md)
* [Configure CN-Operator with service directory](./docs/gcp_service_directory/configure_with_operator.md)

## Contributing

Thank you for interest in contributing to this project.  
Before starting, please make sure you know and agree to our [Code of conduct](./code-of-conduct.md).

1. Fork it
2. Download your fork  
    `git clone https://github.com/your_username/cnwan-operator && cd cnwan-operator`
3. Create your feature branch  
    `git checkout -b my-new-feature`
4. Make changes and add them  
    `git add .`
5. Commit your changes  
    `git commit -m 'Add some feature'`
6. Push to the branch  
    `git push origin my-new-feature`
7. Create new pull request to this repository

## License

CN-WAN Operator is released under the Apache 2.0 license. See [LICENSE](./LICENSE)
