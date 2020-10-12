# Basic Installation

This is the easiest way to install the operator and suitable for most users.

This will install/deploy the latest released version of the operator without
modifying any YAML resource from this repository.

## Requirements

### Files and Services

You need to have the following:

* access to a Kubernetes cluster running at least version `1.11.3` and
[kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)
properly set up
* a Project in [Google Cloud](https://console.cloud.google.com/) with
[Service Directory](https://cloud.google.com/service-directory) enabled
* a [Google Cloud Service Account](https://cloud.google.com/iam/docs/service-accounts)
with at least role `roles/servicedirectory.editor`.

### Software

Please make sure you have the following software installed:

* [Kubectl 1.11.3+](https://kubernetes.io/docs/tasks/tools/install-kubectl/):
  * *Unix/Linux* users with
  [Snap](https://snapcraft.io/docs/installing-snapd):

    ```bash
    snap install kubectl --classic
    ```

  * *MacOs* users with [HomeBrew](https://brew.sh/):

    ```bash
    brew install kubectl
    ```

  * *Windows* users: follow
  [this section](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl-on-windows)
  of the documentation

### Simple Modifications

If you need to do some simple modifications, such as change the docker image or
set a docker pull secret, you can do so by modifying the files in `/deploy`.  
But if you need to do more advanced modifications, then you will have to follow
[Advanced Installation](./advanced_installation.md).

## Configure the operator

Before deploying the operator you will need to configure it.

### Settings

Modify the file `deploy/settings/settings.yaml` with the appropriate values.  
If you haven't already, please read [Configuration](./configuration.md) to
learn how to do this.

### Credentials

Place your Google account in `deploy/settings` and rename it as
`gcloud-credentials.json`.  
Therefore, your `deploy/settings` will contain `settings.yaml` and
`gcloud-credentials.json`.

## Deploy

While you can deploy the operator with plain kubectl commands, CNWAN Operator
comes with scripts that automate such commands for you.

From the root directory of the project, execute

```bash
# Latest official image
./scripts/deploy.sh

# Or another version/your image
./scripts/deploy.sh example.com/repository/image:tag
```

If everything goes fine, CNWAN Operator will run in namespace
`cnwan-operator-system` and deployed to a suitable and available worker node.

If you haven't already, please read [Concepts](./concepts.md) to learn more
about CNWAN Operator.

## Remove

To remove the operator, execute:

```bash
./scripts/remove.sh
```
