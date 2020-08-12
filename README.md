# CNWAN Operator

Register and manage your Kubernetes Services to a Service Registry.

## Table of Contents

* [Overview](#overview)
* [Installing](#installing)
* [Supported Service Registries](#supported-service-registries)
* [Deploying](#deploying)
* [How it works](#how-it-works)
* [Configure the Operator](#configure-the-operator)
  * [The ConfigMap](#the-configmap)
  * [Namespace List Policy](#namespace-list-policy)
  * [Annotations](#annotations)
  * [Service Directory Settings](#service-directory-settigns)
    * [Google Cloud Service Account](#google-cloud-service-account)
* [Uninstalling](#uninstalling)
* [Contributing](#contributing)
* [License](#license)

## Overview

CNWAN Operator is a Kubernetes operator created with [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
that watches for changes in services deployed in your cluster and registers
them to a service registry.

The service registry is populated with only the allowed resources and
properties, as specified from a configuration file.

## Supported Service Registries

Currently, the CNWAN Operator can register and manage Kubernetes services to
Google Cloud's [Service directory](https://cloud.google.com/service-directory).
The project and region must be provided in the ConfigMap, and the service
account file must be provided as a Secret.  
Please follow [this section](#configure-the-operator) to learn how to set up
the operator and provide such files.

## Installing

To install, clone the repository to your pc with the following command:

```bash
git clone https://github.com/CloudNativeSDWAN/cnwan-operator.git
```

Once done, please follow the section on how to [Configure the Operator](#configure-the-operator)
before deploying it to your Kubernetes cluster.

## Deploying

Before deploying the operator, make sure you have read and configured it
properly. Also, you need [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
installed and the [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)
properly set up in order to deploy CNWAN Operator successfully.

First, you need to build and push the docker image to your container registry
of choice. To ease the process up, you can edit the `Makefile` - included in
the root folder of the project - by entering the image repository where
you want to push the image:

```makefile
IMG ?= example.com/username/image:tag
```

Make sure you are properly logged in your container registry of choice before
proceeding. Most of the times, running `docker login <registry>` as documented
[here](https://docs.docker.com/engine/reference/commandline/login/) should be
enough, but we encorage you to read your container registry's official
documentation to know how to do that.  
Build and push the image:

```bash
make docker-build
make docker-push
```

Deploy the operator on your cluster by running:

```bash
make deploy
```

Please refrain from using docker commands directly as the described method
will also test the program before building it.

## How it works

CNWAN Operator implements the [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/):
it is a standalone program that runs in the Kubernetes cluster and extends it
by offering additional functionalities.

Specifically, it watches for changes in [Services](https://kubernetes.io/docs/concepts/services-networking/service/)
and whenever a change is detected, the operator extracts some data out of the
service, i.e. endpoints and annotations, and connects to a [Service Registry](https://auth0.com/blog/an-introduction-to-microservices-part-3-the-service-registry/)
to reflect such changes to it. Example of such changes include new services
deployed, updates to a service's annotations list and deleted services.

Currently, only services of type `LoadBalancer` are supported, and all other
types are ignored by the operator.

## Configure the Operator

### The ConfigMap

From the root folder of the project, navigate to `config/manager`.  
The `configMap`, used to set up the operator, is located inside the
`settings.yaml` file and looks like this:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cnwan-operator-settings
  namespace: system
data:
  settings.yaml: |
    gcloud:
      serviceDirectory:
        region: <region>
        project: <project>
    namespace:
      listPolicy: allowlist
    service:
      annotations:
      - cnwan.io/profile
```

Please follow along to learn how to use this file.

### Namespace List Policy

You can decide which namespaces the operator will work on by configuring
the `listPolicy` parameter inside the `namespace` field.

It accepts the following values:

* `allowlist`: the operator only works on namespaces that are explicitly
allowed and ignore all others. In order to insert a namespace in the allowlist
one must label it `operator.cnwan.io/allowed`, i.e.
`operator.cnwan.io/allowed: yes`.
* `blocklist`: the operator works on *all* namespaces, unless they are inside
the blocklist. To insert a namespace in the blocklist, one must label it
`operator.cnwan.io/blocked`, i.e. `operator.cnwan.io/blocked: yes`.

Please note that these must be *labels*, not annotations.

### Annotations

The `annotations` field, inside `service` field, it's a list of annotations
that will be registered along with the service in form of metadata.  
The operator will look for these annotations whenever a service is subject of
a change, and if it does not include at least one of the accepted values, the
service is simply ignored, or deleted from the registry in case it does not
satisfy the annotations constraints anymore.

Values can also have wildcards. Example of accepted values are:

* Specific values, i.e. `example.prefix.com/name` or `annotation-key`
* Name wildcards, i.e. `example.prefix.com/*`: *all* annotations that have
prefix `example.prefix.com` will be kept and registered, regardless of the
name. For instance, `example.prefix.com/my-name` and
`example.prefix.com/another-name` will both match and therefore be included in
the service's entry as metadata, along with their values.
* Prefix wildcards, i.e. `*/name`, *all* annotations that have name `name`
will be stored and registered, regardless of the prefix.
`example.prefix.com/name` and `another.prefix.com/name` will both match.
* `*/*`: *all* annotations will be registered.

For instance, take a look this service's annotations:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    my.prefix.com/my-name: test-value
    my.prefix.com/another-name: another-value
    another.prefix.com/another-name: yet-another-value
    name-with-no-prefix: simple-value
```

If you allow only the following annotations:

* `my.prefix.com/*`
* `name-with-no-prefix`

The service will be registered with the following metadata:

```yaml
my.prefix.com/my-name: test-value
my.prefix.com/another-name: another-value
name-with-no-prefix: simple-value
```

### Service Directory Settings

Service Directory settings can be configured by changing the values inside
`serviceDirectory`, under `gcloud`.

The Google Cloud project's name and the region must be set by writing the
appropriate values in the `project` and `region` fields respectively.

A note for GKE users: the `region` setting is the default Service Directory
region where services will be registered, *not* your Kubernetes Engine region.
To learn which Service Directory regions are available, read the official
documentation, or you can list them when you create a new namespace from
[Service Directory Console](http://console.cloud.google.com/net-services/service-directory/).

#### Google Cloud Service Account

To properly connect to Service Directory, CNWAN Operator needs a valid [Service
Account](https://cloud.google.com/iam/docs/service-accounts).  
Please follow [this guide](https://cloud.google.com/iam/docs/creating-managing-service-accounts)
to learn more.

The contents of the service account's JSON file must be copied and pasted
to the Kubernetes Secret provided inside the `serviceHandlerSecret.yaml` file,
located inside the `config/manager` folder.

The file needs to look like this:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: service-handler-account
  namespace: system
stringData:
  gcloud-credentials.json: |-
    {
      "type": "service_account",
      "project_id": "my-project",
      "private_key_id": "prive-key-id",
      "private_key": "private-key",
      "client_email": "client-email@example.com",
      "client_id": "1234567890",
      "auth_uri": "https://accounts.google.com/o/oauth2/auth",
      "token_uri": "https://oauth2.googleapis.com/token",
      "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
      "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/name"
    }
```

Make sure to respect `YAML` identation properties, as the copied content
must have some tabs/spaces to be correctly included under
`gcloud-credentials.json`, the same way as the above example.

## Uninstalling

To remove the operator from your Kubernetes cluster, navigate to the root
directory of the project and execute:

```bash
make remove
```

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

CNWAN Operator is released under the Apache 2.0 license. See [LICENSE](./LICENSE)
