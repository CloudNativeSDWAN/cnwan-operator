# Configure the Operator

This section will guide you through the steps you need to take to configure the CN-WAN Operator.

## Table of Contents

* [Format](#format)
* [Watch namespaces by default](#watch-namespaces-by-default)
* [Allow Annotations](#allow-annotations)
* [Cloud Metadata](#cloud-metadata)
* [Service registry settings](#service-registry-settings)
* [Deploy settings](#deploy-settings)
* [Update settings](#update-settings)

## Format

The CN-WAN Operator can be configured with the following YAML format.

```yaml
watchNamespacesByDefault: false
serviceAnnotations: []
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
  awsCloudMap:
    defaultRegion: <region>
cloudMetadata:
  network: auto
  subNetwork: auto
```

## Watch namespaces by default

The operator will observe service events only on namespaces that are *watched*, and to do so you need to explicitly label namespaces with the reserved `operator.cnwan.io/watch` label key.

`watchNamespacesByDefault` will tell the operator what to do when such label is not found: if it does not exist or is false, then the operator will ignore the namespace by default. Otherwise it will watch events inside it.

if you haven't already, please take a look at [this section](./concepts.md#watch-namespaces) to learn more about this concept.

## Allow Annotations

The operator will not register every annotation as metadata from a Kubernetes Service, but will only do so with the ones you have explicitly allowed.

if you haven't already, please take a look at [Metadata](./concepts.md#metadata), [Allowed Annotations](./concepts.md#allowed-annotations) and [Annotations vs Labels](./concepts.md#annotations-vs-labels) to learn more.

You can allow annotations by setting up `serviceAnnotations` in the configuration. For example:

```yaml
serviceAnnotations:
  - version
  - example.com/purpose
```

Or you may like this format better:

```yaml
serviceAnnotations: [version, example.com/purpose]
```

Values can also have wildcards. Example of accepted values are:

* Specific values, i.e. `example.prefix.com/name` or `annotation-key`
* Name wildcards, i.e. `example.prefix.com/*`: *all* annotations that have prefix `example.prefix.com` will be kept and registered, regardless of the name. For instance, `example.prefix.com/my-name` and `example.prefix.com/another-name` will both match and therefore be included in the service's entry as metadata, along with their values.
* Prefix wildcards, i.e. `*/name`, *all* annotations that have name `name` will be stored and registered, regardless of the prefix. `example.prefix.com/name` and `another.prefix.com/name` will both match.
* `*/*`: *all* annotations will be registered. We discourage you from using this value, as you may potentially expose sensitive information about the service.

For instance, take a look at this service's annotations:

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

Finally, if you leave this empty - as `serviceAnnotations: []`, then no service will match this and, therefore, no service will be registered.

## Cloud Metadata

Cloud Metadata can be registered automatically through the `cloudMetadata` setting.

You can provide manual values by entering the information you want like this:

```yaml
cloudMetadata:
  network: my-vpc-id
  subNetwork: my-subnet-id
```

or automatically as:

```yaml
cloudMetadata:
  network: auto
  subNetwork: auto
```

and the Operator will try to detect such information on its own. Note that automatic feature is only supported for *GKE* and for the other platforms you will have to write that information manually until they will be supported as well.

You can remove a field, e.g. `subNetwork`, from the settings if you don't want that to be registered.

These values will be registered on a service metadata as:

```yaml
cnwan.io/network: <name-or-id>
cnwan.io/sub-network: <name-or-id>
```

Additionally, `cnwan.io/platform: <name>` will also be included if the operator detects you are running in a managed cluster.

## Service registry settings

Under `serviceRegistry` you define which service registry to use and how the operator should connect to it or manage its objects.

As of now, only one of `etcd`, `gcpServiceDirectory` or `awsCloudMap` is allowed, and therefore you should remove the one that you don't use. Please follow one of the following guides to learn how to configure the Operator with the chosen service registry:

* [etcd](./etcd/operator_configuration.md)
* [Service Directory](./gcp_service_directory/configure_with_operator.md)
* [Cloud Map](./aws_cloud_map/operator_configuration.md)

## Deploy settings

To deploy these settings you will have to follow the [installation guide](./install.md)

## Update settings

To update the settings, you can run

```bash
kubectl edit configmap cnwan-operator-settings -n cnwan-operator-system
```

This will open your default editor and you will be able to edit the settings inline.

If successful, you will have to restart the operator for it to be able to acknowledge the changes:

```bash
# For Kubernetes 1.15+
kubectl rollout restart deployment cnwan-operator -n cnwan-operator-system
```

In case your Kubernetes version is lower, than you will have to either delete the pod or scale down the deployment:

```bash
NAME=$(kubectl get pods -o jsonpath='{.items[0].metadata.name}' -n cnwan-operator-system)
kubectl delete pod $NAME
```
