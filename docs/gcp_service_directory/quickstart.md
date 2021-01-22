# Quickstart

Curious to see CN-WAN Operator in action but feeling lazy about learning how it does stuff? Follow this guide!

## Requirements

To run this, make sure you have:

* Access to a Kubernetes cluster running at least version `1.11.3`
  * [Minikube](https://kubernetes.io/docs/setup/learning-environment/minikube/) is also fine.
* [Kubectl 1.11.3+](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* A Project in [Google Cloud](https://console.cloud.google.com/) with [Service Directory](https://cloud.google.com/service-directory) enabled
* A [Google Cloud Service Account](https://cloud.google.com/iam/docs/service-accounts) with at least role `roles/servicedirectory.editor`.

Additionally:

* [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) properly set up.
* Your cluster is able to perform outbound HTTP/S requests successfully.
* Your cluster supports [LoadBalancer](./concepts.md#supported-service-types) type of services.

## Let's go

### 1 - Clone the project

```bash
git clone https://github.com/CloudNativeSDWAN/cnwan-operator.git
cd ./cnwan-operator
```

### 2 - Deploy a test service

We will suppose you are deploying an application on your cluster where your employees can log in and watch training videos.

Run this to deploy a new namespace and a service in that namespace:

```bash
cat <<EOF | kubectl create -f -
kind: Namespace
apiVersion: v1
metadata:
    name: training-app-namespace
    labels:
        purpose: "test"
        operator.cnwan.io/allowed: "yes"
---
kind: Service
apiVersion: v1
metadata:
    name: web-training
    namespace: training-app-namespace
    labels:
        app: "training"
    annotations:
        version: "2.1"
        traffic-profile: "standard"
spec:
    ports:
        - name: port80
          protocol: TCP
          port: 80
          targetPort: 8080
    selector:
        app: "training"
    type: LoadBalancer
EOF
```

Please notice that the namespace has this label: `operator.cnwan.io/allowed: yes` which inserts the namespace in the opeartor's [allowlist](./concepts.md#namespace-lists). Also notice that the service has annotations that will be registered as [metadata](./concepts.md#metadata):

```yaml
annotations:
   traffic-profile: standard
```

Now verify that the namespace is there:

```bash
kubectl get ns

NAME                   STATUS   AGE
training-app-namespace          Active   1h
```

Verify that the service is there and has an IP:

```bash
kubectl get service -n training-app-namespace

NAME                   TYPE           CLUSTER-IP       EXTERNAL-IP    PORT(S)                       AGE
web-training           LoadBalancer   10.11.12.13      20.21.22.23    80:32058/TCP                  1h
```

If you see `<none>` or `<pending>` under `EXTERNAL-IP` you either have to wait to see an IP there or your cluster doesn't support [LoadBalancer](./concepts.md#supported-service-types).

It doesn't really matter that there is no pod backing this service for now, as this is just a test. Of course, in a real world scenario you should make sure a pod is there.

### 3 - Provide the service account

Navigate to the root directory and place your service account to `deploy/settings`, with name `gcloud-credentials.json`. So, you will have `deploy/settings/gcloud-credentials.json`.

### 4 - Provide settings

From the root directory navigate to `deploy/settings` and modify the file `settings.yaml` to look like this - please provide appropriate values in place of `<gcloud-project>` and `<gcloud-region>`:

```yaml
gcloud:
  serviceDirectory:
    region: <gcloud-region>
    project: <gcloud-project>
namespace:
  listPolicy: allowlist
service:
  annotations:
  - traffic-profile
  - version
```

Please notice the values inside `annotations`:

```yaml
  annotations:
  - traffic-profile
  - version
```

This means that the operator will register `traffic-profile` as metadata if it finds it among a [service's annotations list](./concepts.md#allowed-annotations).

### 5 - Deploy the operator

From the root directory of the project, execute

```bash
./scripts/deploy.sh
```

### 6 - See it on Service Directory

Now, log in to Service Directory from the google cloud console and you will see a namespace that has the same name as the Kubernetes namespace where that service was found in.

If you click on it, you will see a service: its metadata contain `traffic-profile: standard`. It will also contain an endpoint with data about the port and the address.

### 7 - Update metadata

Now you're basically done, but you can follow these additional steps to see more of the operator in action.

Suppose you made a mistake: this is a training application where your employees will follow video tutorials. Therefore, its kind of traffic - or, *profile*, must be `video`.

Execute:

```bash
kubectl annotate service web-training traffic-profile=video --overwrite -n training-app-namespace
```

The operator has updated the metadata in Service Directory accordingly.

### 8 - Add new metadata

Suppose you have a CI/CD pipeline that for each PR builds a container with a new tag. Also, it updates the service that serves the pods running that container by specifying the new version. Today, you will be that pipeline:

```bash
kubectl annotate service web-training version=2.2 -n training-app-namespace
```

Once again, log in to Service Directory and see how the metadata for that service have changed accordingly.

## Where to go from here

Well, that's it for a quickstart. Now we encourage you to learn more about CN-WAN Operator by taking a look at the [docs](./).

Also, make sure you read the [official documentation of CN-WAN](https://github.com/CloudNativeSDWAN/cnwan-docs) to learn how you can apply this simple quickstart to a real world scenario.

## Clean up

From the root directory of the project, run

```bash
./scripts/remove.sh
```
