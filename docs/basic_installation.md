# Basic Installation

This is the easiest way to install the operator and suitable for most users.

This will install/deploy the latest released version of the operator without modifying any YAML resource from this repository.

## Requirements

### Files and Services

You need to have the following:

* access to a Kubernetes cluster running at least version `1.11.3` and [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) properly set up

Depending on the service registry you chose:

* For Google Service Directory:
  * a Project in [Google Cloud](https://console.cloud.google.com/) with [Service Directory](https://cloud.google.com/service-directory) enabled
  * a [Google Cloud Service Account](https://cloud.google.com/iam/docs/service-accounts) with at least role `roles/servicedirectory.editor`.
* For etcd:
  * a working and reachable etcd cluster
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

### Simple Modifications

If you need to do some simple modifications, such as change the docker image or set a docker pull secret, you can do so by modifying the files in `/deploy`. But if you need to do more advanced modifications, then you will have to follow [Advanced Installation](./advanced_installation.md).

## Configure the operator

Before deploying the operator you will need to configure it.

### Settings

Modify the file `deploy/settings/settings.yaml` with the appropriate values. If you haven't already, please read [Configuration](./configuration.md) to learn how to do this.

## Deploy

While you can deploy the operator with plain kubectl commands, CN-WAN Operator comes with scripts that automate such commands for you.

To deploy the operator with the provided script, you will have to execute `deploy.sh` as such:

```bash
./scripts/deploy.sh <service-registry>
```

If you want to use your own image, you'll need to provide `--img` flag, for example:

```bash
./scripts/deploy.sh etcd --img example.com/repository/image:tag
```

Follow the two sections below according to the service registry you chose and, after that and if everything goes fine, CN-WAN Operator will run in namespace `cnwan-operator-system` and deployed to a suitable and available worker node.

If you haven't already, please read [Concepts](./concepts.md) to learn more about CN-WAN Operator.

### With etcd

From the root directory of the project, execute

```bash
./scripts/deploy.sh etcd --username <username> --password <password>

# Without authentication mode
./scripts/deploy.sh etcd
```

### With Google Service Directory

Place your Google account in `deploy/settings` and rename it as `gcloud-credentials.json`. Therefore, your `deploy/settings` will contain `settings.yaml` and `gcloud-credentials.json`.

Now run

```bash
./scripts/deploy.sh servicedirectory
```

## Remove

To remove the operator, execute:

```bash
./scripts/remove.sh
```
