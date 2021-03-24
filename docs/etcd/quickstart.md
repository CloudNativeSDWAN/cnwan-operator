# Quickstart

Follow this guide if you want to see how the CN-WAN Operator automatically connects and manages a service registry on top of etcd.

## Requirements

To run this, make sure you have:

* A working etcd cluster: you can follow [this guide](./demo_cluster_setup.md) to create a **demo** cluster for this quickstart
* Access to a Kubernetes cluster running at least version `1.11.3`
  * [Minikube](https://kubernetes.io/docs/setup/learning-environment/minikube/) is fine.
* [Kubectl 1.11.3+](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Also, please make sure that:

* your [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) is properly set up.
* your cluster supports [LoadBalancer](../concepts.md#supported-service-types) type of services.

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

Please notice that the namespace has this label: `operator.cnwan.io/allowed: yes` which inserts the namespace in the opeartor's [allowlist](../concepts.md#namespace-lists). Also notice that the service has annotations that will be registered as [metadata](../concepts.md#metadata):

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

If you see `<none>` or `<pending>` under `EXTERNAL-IP` you either have to wait to see an IP there or your cluster doesn't support [LoadBalancer](../concepts.md#supported-service-types).

It doesn't really matter that there is no pod backing this service for now, as this is just a test. Of course, in a real world scenario you should make sure a pod is there.

### 3 - Provide settings

From the root directory navigate to `deploy/settings` and modify the file `settings.yaml` to look like this - please provide appropriate values for `host` and `port` keys with your etcd cluster's addresses:

```yaml
namespace:
  listPolicy: allowlist
service:
  annotations:
  - traffic-profile
  - version
serviceRegistry:
  etcd:
    authentication: WithUsernameAndPassword
    endpoints:
    - host: <host-1>
      port: <port-1>
    - host: <host-2>
      port: <port-2>
```

If you have followed our [demo cluster](./demo_cluster_setup.md) guide abd supposing the address you chose is `10.10.10.10`, the your `endpoints` setting just looks like this:

```yaml
endpoints:
- host: 10.10.10.10
```

Please notice the values inside `annotations`:

```yaml
  annotations:
  - traffic-profile
  - version
```

This means that the operator will register `traffic-profile` as metadata if it finds it among a [service's annotations list](../concepts.md#allowed-annotations).

**Important**: if you **don't** have [authentication mode](./demo_cluster_setup.md#make-it-more-secure) you can remove `authentication: WithUsernameAndPassword` entirely. We encourage you to read and learn more about [etcd settings](./operator_configuration.md).

### 4 - Deploy the operator

From the root directory of the project, execute one of the following lines:

```bash
# If you have username and password for etcd
./scripts/deploy.sh etcd --username <username> --password <password>

# If you don't have username and password for etcd
./scripts/deploy.sh etcd
```

### 5 - See it on etcd

Log in to etcd and look at data there with `etcdctl` - modify `host:port` and `user` accordingly:

```bash
etcdctl --endpoints http://host:port --user user:password get /service-registry/ --prefix
```

`/service-registry` is the [prefix](./concepts.md#prefix) that all service registry objects will have on their key. This is the default value and it's there because we didn't [configure CN-WAN operator](./operator_configuration.md) with a different prefix.

Now, watch for changes there:

```bash
etcdctl --endpoints http://host:port --user user:password watch /service-registry/ --prefix
```

### 6 - Update metadata

Now you're basically done, but you can follow these additional steps to see more of the operator in action.

Suppose you made a mistake: this is a training application where your employees will follow video tutorials. Therefore, its kind of traffic - or, *profile*, must be `video`.

Execute:

```bash
kubectl annotate service web-training traffic-profile=video --overwrite -n training-app-namespace
```

The operator has updated the metadata in etcd accordingly.

### 7 - Add new metadata

Suppose you have a CI/CD pipeline that for each PR builds a container with a new tag. Also, it updates the service that serves the pods running that container by specifying the new version. Today, you will be that pipeline:

```bash
kubectl annotate service web-training version=2.2 -n training-app-namespace --overwrite
```

Once again, you will see that the metadata for that service have changed accordingly in etcd.

## Where to go from here

Well, that's it for a quickstart. Now we encourage you to learn more about CN-WAN Operator by taking a look at the [CN-WAN Operator docs](../../README.md#documentation) and [etcd docs](../../README.md#etcd-documentation).

Also, make sure you read the [official documentation of CN-WAN](https://github.com/CloudNativeSDWAN/cnwan-docs) to learn how you can apply this simple quickstart to a real world scenario.

## Clean up

From the root directory of the project, run

```bash
./scripts/remove.sh
```
