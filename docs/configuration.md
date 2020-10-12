# Configure the Operator

This section will guide you through the steps you need to take to configure
the CNWAN Operator.

## Table of Contents

* [Format](#format)
* [Google Cloud Settings](#google-cloud-settings)
* [Set the Namespace List Policy](#set-the-namespace-list-policy)
* [Allow Annotations](#allow-annotations)
* [Deploy](#deploy)
* [Update](#update)

## Format

The CNWAN Operator can be configured with the following YAML format.

```yaml
gcloud:
  serviceDirectory:
    region: <region>
    project: <project>
namespace:
  listPolicy: allowlist
service:
  annotations: []
```

## Google Cloud Settings

Under `gcloud` you can specify Google Cloud data. For example, you can specify
the project and the region where Service Directory is enabled and you want to
be managed.

You can modify `region` and `project` with the appropriate values.

For example:

```yaml
gcloud:
  serviceDirectory:
    region: us-central1
    project: this-is-my-project
```

## Set the Namespace List Policy

The operator will only monitor services that belong to a namespace that you
have explicitly allowed.

if you haven't already, please take a look at
[this section](./concepts.md#namespace-list-policy) to learn more about
the *default namespace list policy*.

To set the list policy, change `listPolicy` value to either `allowlist`
or `blocklist` like so:

```yaml
namespace:
  listPolicy: allowlist
```

## Allow Annotations

The operator will not register every annotation as metadata from a Kubernetes
Service, but will only do so with the ones you have explicitly allowed.

if you haven't already, please take a look at
[Metadata](./concepts.md#metadata),
[Allowed Annotations](./concepts.md#allowed-annotations) and
[Annotations vs Labels](./concepts.md#annotations-vs-labels) to learn more.

You can allow annotations by setting up `service.annotations` in the
configuration. For example:

```yaml
service:
  annotations:
  - version
  - example.com/purpose
```

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
* `*/*`: *all* annotations will be registered. We discourage you from using
this value, as you may potentially expose sensitive information about the
service.

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

Finally, if you leave this empty - as `annotations: []`, then no service will
match this and, therefore, no service will be registered.

## Deploy

To deploy these settings you will have to follow either
[Basic Installation](./basic_installation.md) or
[Advanced Installation](./advanced_installation.md).

## Update

To update the settings, you can run

```bash
kubectl edit configmap cnwan-operator-settings -n cnwan-operator-system
```

This will open your default editor and you will be able to edit the settings
inline.

If successful, you will have to restart the operator for it to be able to
acknowledge the changes:

```bash
# For Kubernetes 1.15+
kubectl rollout restart deployment cnwan-operator-controller-manager -n cnwan-operator-system
```

In case your Kubernetes version is lower, than you will have to either delete
the pod or scale down the deployment:

```bash
NAME=$(kubectl get pods -o jsonpath='{.items[0].metadata.name}' -n cnwan-operator-system)
kubectl delete pod $NAME
```
