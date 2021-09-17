# Install

You can deploy the operator in two ways:

* *helm charts*: please follow our [official helm chart repository](https://github.com/CloudNativeSDWAN/cnwan-helm-charts).
* *bash scripts*: continue reading this guide.

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

## Configure the operator

Before deploying the operator you will need to configure it.

### Settings

Modify the file `artifacts/deploy/settings/settings.yaml` with the appropriate values. If you haven't already, please read [Configuration](./configuration.md) to learn how to do this.

## Deploy

While you can deploy the operator with plain kubectl commands, CN-WAN Operator comes with scripts that automate such commands for you.

Before continuing, if you also have other files that you want to be deployed, you may want to follow the [Adding resources](#adding-resources) section.

To deploy the operator with the provided script, you will have to execute `deploy.sh` as such:

```bash
./scripts/deploy.sh etcd|servicedirectory
```

If you want to use your own image, you'll need to provide `--image` flag, for example:

```bash
./scripts/deploy.sh etcd --image example.com/repository/image:tag
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

## Adding resources

If you want to add resources or make modifications to the CN-WAN Operator -- i.e. if you want to contribute -- we recommend you to do so by following the same coding and formatting style as provided in the existing files.

This means that you will just need to make all the necessary modifications in the existing yaml files in `/artifacts`, i.e. add `pullImageSecrets` in `/artifacts/deploy/07_deployment.yaml.tpl` and then later run `/scripts/deploy.sh` as usual.

Instead, if you want to add files that did not exist, you can do that by doing one of the following steps:

* add those yamls in `/artifacts/deploy/other`
* add the yamls in `/artifacts/deploy` and modify `/scripts/deploy.sh` accordingly by adding those files yourself, for example, at the bottom of `/scripts/deploy.sh`:

    ```bash
    # ... other content
    kubectl create -f $DEPLOY_DIR/<my_file-1.yaml>,$DEPLOY_DIR/<my-file-2.yaml>
    kubectl create -f $DEPLOY_DIR/07_deployment_generated.yaml
    ```

* manually deploy those files via `kubectl` and later re-start the operator via `kubectl rollout restart deployment cnwan-operator-controller-manager`.

Make sure those resources have the appropriate `namespace` in case you need them to be deployed in the same namespace as the operator, which is `cnwan-operator-system` by default, and to also run the appropriate `kubectl delete` either in `/scripts/remove.sh` -- in case you choose the second method -- or manually.

Although the first method is recommended for most cases, you should use the second one when contributing, in which case we kindly ask you to submit an issue or a discussion so that the owners can better help you out, i.e. recommend you how to modify the scripts, how to organize your files, naming and code conventions, etc. Finally, please read our [contributing guide](../README.md#contributing) and [code of conduct](../code-of-conduct.md) as well.
