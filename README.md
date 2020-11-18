![GitHub](https://img.shields.io/github/license/CloudNativeSDWAN/cnwan-operator)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/CloudNativeSDWAN/cnwan-operator)
![Kubernetes version](https://img.shields.io/badge/kubernetes-1.11.3%2B-blue)
![GitHub Workflow Status](https://img.shields.io/github/workflow/status/CloudNativeSDWAN/cnwan-operator/Test)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/CloudNativeSDWAN/cnwan-operator)
![Docker Image Version (latest by date)](https://img.shields.io/docker/v/cnwan/cnwan-operator?label=docker%20version)

# CN-WAN Operator

Register and manage your Kubernetes Services to a Service Registry.

The CN-WAN Operator is part of the Cloud Native SD-WAN (CN-WAN) project.
Please check the
[CN-WAN documentation](https://github.com/CloudNativeSDWAN/cnwan-docs) for the
general project overview and architecture.
You can contact the CN-WAN team at [cnwan@cisco.com](mailto:cnwan@cisco.com).

## Overview

CN-WAN Operator is a Kubernetes operator created with [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
that watches for changes in services deployed in your cluster and registers
them to a service registry so that clients can later discover all registered
services and know how to connect to them properly.

The service registry is populated with only the allowed resources and
properties, as specified from a configuration file.

## Supported Service Registries

Currently, the CN-WAN Operator can register and manage Kubernetes services to
Google Cloud's [Service Directory](https://cloud.google.com/service-directory).
The project and region must be provided in the `ConfigMap`, and the service
account file must be provided as a `Secret`.  
Please follow [this section](#configure-the-operator) to learn how to set up
the operator and provide such files.

## Try It Out

If you want to quickly see how CN-WAN Operator works, you can follow this simple
step by step [quickstart](./docs/quickstart.md) guide.

## Documentation

* [Concepts](./docs/concepts.md)
* [Configuration](./docs/configuration.md)
* [Basic Installation](./docs/basic_installation.md)
* [Advanced Installation](./docs/advanced_installation.md)

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
