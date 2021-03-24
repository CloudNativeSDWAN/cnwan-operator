# Configure CN-WAN Operator with etcd

This short guide is focused on configuring the CN-WAN Operator to use and configure etcd as a service registry.

## Settings format

The included directory `deploy/settings` contains a `settings.yaml` for you to modify with the appropriate values.

For your convenience, here is how the settings for the CN-WAN Operator looks like:

```yaml
namespace:
  listPolicy: allowlist
service:
  annotations: []
serviceRegistry:
  etcd:
    prefix: <prefix>
    authentication: <your-authentication-type>
    endpoints:
    - host: <host-1>
      port: <port-1>
    - host: <host-2>
      port: <port-2>
  gcpServiceDirectory:
    defaultRegion: <region>
    projectID: <project>
```

We will only cover etcd settings here, so you can go ahead and remove the whole `gcpServiceDirectory` settings:

```yaml
namespace:
  listPolicy: allowlist
service:
  annotations: []
serviceRegistry:
  etcd:
    prefix: <prefix>
    authentication: <your-authentication-type>
    endpoints:
    - host: <host-1>
      port: <port-1>
    - host: <host-2>
      port: <port-2>
```

`namespace` and `service` settings are covered in the [main documentation](../configuration.md). Let's now only focus on `serviceRegistry` options.

## etcd settings

### Prefix

Set the `prefix` to whatever value you want/need and all object keys will start with it. In case you omit or leave it empty the CN-WAN Operator will use `/service-registry` as a default prefix. If you don't want any prefix just write `/`.

If you haven't already, please read our [etcd concepts documentation](./concepts.md) to learn more about keys and prefixes.

### Authentication

If present, `authentication` field accepts two values:

* `WithUsernameAndPassword`
* `WithTLS`

If you don't include this CN-WAN Operator will authenticate to etcd as a guest user. Although the security aspects of this are out of scope of this guide, we recommend you to enable authentication mode: we have a guide on just that [here](./demo_cluster_setup.md#make-it-more-secure).

#### Authenticate with username and password

If you decide to authenticate with username and password, you need to set `WithUsernameAndPassword` value, like so:

```yaml
authentication: WithUsernameAndPassword
```

With this, the CN-WAN Operator will expect a `Secret` to exist in your cluster called `cnwan-operator-etcd-credentials` in the same namespace where the Operator is running (`cnwan-operator-system`).

To create this secret, you execute the following command - please edit `<username>` and `<password>` accordingly:

```bash
kubectl create secret generic cnwan-operator-etcd-credentials \
-n cnwan-operator-system \
--from-literal=username=<username> \
--from-literal=password=<password>
```

This will create the secret on your cluster.

CN-WAN Operator considers the absence of this secret as an error and terminates the execution. Don't worry, though: you can still deploy the secret and, when it will be re-scheduled by *Kubernetes*, it will find it and work as expected.

#### Authenticate with TLS

If you decide to authenticate with *TLS*, you need to set `WithTLS` value, like so:

```yaml
authentication: WithTLS
```

Although, please keep in mind that this feature is not available yet. Stay tuned with CN-WAN Operator to learn when this feature will be introduced.

### Endpoints

`endpoints` is a list of `host`s and `port`s of your etcd cluster. Although it is out of the scope of this guide, you should keep in mind that to enforce reliability and resilience you should have multiple etcd nodes, ideally an odd number of nodes to increase failure tolerance. For more information, please read etcd's [official documentation](https://etcd.io/) and its *FAQ*.

This being said, you don't need enter *all* the nodes there of course, but make sure your write a few.

If you are using etcd's default port `2379`, you can go ahead and omit `port`.

If you followed our [Demo Cluster Setup](./demo_cluster_setup.md) you would have only one endpoint, which is the address you chose there -- `ETCD_IP`.

## Full example

### Example 1

In this example, you are telling the CN-WAN Operator:

* to authenticate to etcd as a **guest**
* use `/clusters/cluster-1/service-registry` as a prefix
* `10.10.10.10`, `10.10.10.11`, `10.10.10.12` as endpoints (the addresses exposed by etcd servers), all with default port (`2379`)

Here is the settings example - we omit `namespace` and `service` settings for brevity:

```yaml
namespace: ...
service: ...
serviceRegistry:
  etcd:
    prefix: /clusters/cluster-1/service-registry
    endpoints:
    - host: 10.10.10.10
    - host: 10.10.10.11
    - host: 10.10.10.12

```

### Example 2

In this example, you are telling the CN-WAN Operator:

* to authenticate to etcd with a username and password that it will find on `Secret` `cnwan-operator-etcd-credentials`
* use the default prefix (`/service-registry`)
* use `10.10.10.10` with port `3344`, `10.10.10.11` on port `4433`,  and `10.10.10.12` on default port (`2379`)

Here is the settings example - we omit `namespace` and `service` settings for brevity:

```yaml
namespace: ...
service: ...
serviceRegistry:
  etcd:
    authentication: WithUsernameAndPassword
    endpoints:
    - host: 10.10.10.10
      port: 3344
    - host: 10.10.10.11
      port: 4433
    - host: 10.10.10.12
```
