# Advanced Installation

This installation is for advanced users that want to contribute to the project and/or add new resources or modify existing resources.

If you want to contribute only to the operator's code or otherwise don't have to do any substantial resource modification, then please follow [Basic Installation](./basic_installation.md).

This will require some additional dependencies and a knowledge of [Kustomize](https://kubernetes-sigs.github.io/kustomize/guides/).

If you do want to contribute, please follow [Contributing](../README.md#contributing) and our [Code of Conduct](../code-of-conduct.md) before doing so.

## Requirements

### Files and Services

You need to have the following:

* access to a Kubernetes cluster running at least version `1.11.3` and [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) properly set up.

Depending on the service registry you chose:

* For Google Service Directory:
  * a Project in [Google Cloud](https://console.cloud.google.com/) with [Service Directory](https://cloud.google.com/service-directory) enabled
  * a [Google Cloud Service Account](https://cloud.google.com/iam/docs/service-accounts) with at least role `roles/servicedirectory.editor`.
* For etcd:
  * a working and reachable etcd cluster, if you don't have any [follow this guide](./etcd/demo_cluster_setup.md)
  * optional: user credentials for etcd

### Software

Please make sure you have the following software installed:

* [Kubectl 1.11.3+](https://kubernetes.io/docs/tasks/tools/install-kubectl/):
  * *Unix/Linux* users with [Snap](https://snapcraft.io/docs/installing-snapd):

    ```bash
    snap install kubectl --classic
    ```

  * *MacOs* users with [HomeBrew](https://brew.sh/):

    ```bash
    brew install kubectl
    ```

  * *Windows* users: follow [this section](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl-on-windows) of the documentation
* [Make](https://www.gnu.org/software/make/), which will automate some steps:
  * *Unix/Linux and MacOS*: already pre-installed.
  * *Windows* users: download the binaries from [this page](http://gnuwin32.sourceforge.net/packages/make.htm).
* [Golang 1.13+](https://golang.org/doc/install) to build the project. Follow the link to learn how to install it for any system.
* [Docker 17.03+](https://www.docker.com/get-started) for building and pushing the operator's container images.
  * *Unix/Linux* users with [Snap](https://snapcraft.io/docs/installing-snapd):

  ```bash
  sudo snap install docker
  ```

  * *MacOs* users: [Docker Desktop for Mac](https://hub.docker.com/editions/community/docker-ce-desktop-mac/)
  * *Windows* users: [Docker Desktop for Windows](https://hub.docker.com/editions/community/docker-ce-desktop-windows/)

* [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder#installation):
  * *MacOs* users with [HomeBrew](https://brew.sh/):

  ```bash
  brew install kubebuilder
  ```

  * *Other systems*: follow [this section](https://book.kubebuilder.io/quick-start.html#installation) on the documentation.

### Optional: New YAML files

If you are not adding any new Kubernetes resources, such as `Secret`s, `Deployment`s, `Service`s, etc., you can skip this section and go directly to [Configure the operator](#configure-the-operator).

Note that this is different from `CRD`s, as the CN-WAN Operator does not have any custom resources.

As a reminder, if you are adding resources to CN-WAN Operator to contribute to the project, please discuss the changes you want to make with the CN-WAN Operator [OWNERS](../OWNERS.md) by opening a new [issue](https://github.com/CloudNativeSDWAN/cnwan-operator/issues) or by email prior to make a pull request.

Finally, if you just want to do simple modifications, like set a docker pull secret, you should modify files inside `deploy` and follow [Basic Installation](./basic_installation.md).

#### Directories organization

You will have to put the `YAML` files in one of the sub-directories of `/config`: if you are modifying/adding resources just for your own sake, then you can place them in whichever folder you want, so long as you also modify `kustomazion.yaml` accordingly, as specified below.

Instead, in case you are adding files for the project, we ask you to place files depending on the `Kind` of such resources: i.e. `Role`s in `rbac`, `WebHook`s in `webhook` and everything else in `manager`.

Modify the `kustomization.yaml` file by adding the file you just placed. For example, take a look at `config/manager/kustomization.yaml`:

```yaml
resources:
- manager.yaml
- settings.yaml

# Remove this if you're not using servicedirectory
- serviceHandlerSecret.yaml
patchesStrategicMerge:
- patch.yaml
```

If you are adding a new `Service`, add its file name without path under `resources:` the same way you see above. Specify any modification you want to do on resources, by adding your patch under `patchesStrategicMerge:`.

Please take a look at [this guide](https://kubernetes-sigs.github.io/kustomize/guides/) to learn how to use Kustomize in case this looks too obscure.

## Configure the operator

Before deploying the operator you will need to configure it.

### Settings

Modify the file `config/manager/settings.yaml` with the appropriate values. You will need to modify what's below `settings.yaml: |` and follow [Configuration](./configuration.md) if you haven't already.

### Credentials

*Skip this step if you are not using Google Service Directory.*

Copy the contents of you Service Account and paste to `config/manager/serviceHandlerSecret.yaml` below `gcloud-credentials.json: |-`.

The file must look like this:

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

Please **double-check indentation**: if invalid, it will violate yaml parsing rules and treated as an empty string. Make sure it is as above.

## Build the Operator

First, you need to build and push the docker image to your container registry of choice. To ease the process up, you can edit the `Makefile` - included in the root folder of the project - by entering the image repository where you want to push the image:

```makefile
IMG ?= example.com/username/image:tag
```

Make sure you are properly logged in your container registry of choice before proceeding. Most of the times, running `docker login <registry>` as documented [here](https://docs.docker.com/engine/reference/commandline/login/) should be enough, but we encourage you to read your container registry's official documentation to know how to do that. Build and push the image:

```bash
# Build & Push
make docker-build docker-push
```

## Deploy

Deploy the operator on your cluster by running the command below from the root directory of the project:

```bash
make custom-deploy
```

The operator will be first tested and, if successful, installed in one of the available and suitable worker nodes of your cluster.

If you haven't already, please read [Concepts](./concepts.md) to learn more about CN-WAN Operator.

## Remove

To remove the operator from your cluster, execute:

```bash
make custom-remove
```
