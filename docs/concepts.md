# Concepts

## Table of Contents

* [How it Works](#how-it-works)
* [Supported Service Types](#supported-service-types)
* [Metadata](#metadata)
* [Annotations vs Labels](#annotations-vs-labels)
* [Ownership](#ownership)
* [Enable namespaces](#enable-namespaces)
* [Allowed Annotations](#allowed-annotations)
* [Cloud Metadata](#cloud-metadata)
* [Deploy](#deploy)

## How it Works

CN-WAN Operator implements the [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/): it is a standalone program that runs in the Kubernetes cluster and extends it by offering additional functionalities.

Specifically, it watches for changes in [Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/) and whenever a change is detected, the operator extracts some data out of the service, i.e. endpoints and annotations, and connects to a [Service Registry](https://auth0.com/blog/an-introduction-to-microservices-part-3-the-service-registry/) to reflect such changes to it. Example of such changes include new services deployed, updates to a service's annotations list and deleted services.

## Supported Service Types

Currently, only services of type `LoadBalancer` are supported, and all other types are ignored by the operator.

Please make sure your cluster supports load balancers before deploying the operator: most managed Kubernetes platforms do support them, but in case you are not running a managed Kubernetes you may use [MetalLB](https://metallb.universe.tf/) or explore other load balancer solutions.

## Metadata

When the CN-WAN Operator registers/modifies a service in the service registry, it will also register some metadata with it, if the service registry allows it. Think of metadata as a collection of `key: value` pairs that provide more information about the service. For example, you may want to label a service with metadata `version: v.2.2.1`.

You can define the metadata you wish to be registered in a service by **annotating** the corresponding Kubernetes Service. For example:

```bash
kubectl annotate service my-service version=2.1
```

The operator will see this annotation and, [if you enable it](#allowed-annotations), it will be kept and inserted among the service's metadata when it is published in the service registry.

More information and examples [here](./service_registry.md).

## Annotations vs Labels

Let's further elaborate the *Metadata* section and specify why we treat *annotations* as metadata instead of doing that with *labels*.

The CN-WAN Operator reads *annotations* and not *labels* because they are the closest to metadata: let's take a look at how Kubernetes defines [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) and [labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/):

> **Labels** can be used to select objects and to find collections of objects that satisfy certain conditions.
>
> In contrast, **annotations** are not used to identify and select objects.
>
> **Labels** are intended to be used to specify identifying attributes of objects that are meaningful and relevant to users.
>
> Non-identifying information should be recorded using **annotations**.

And, one use case for what you can store in annotations:

> Build, release, or image information like timestamps, release IDs, git branch, PR numbers, image hashes, and registry address.

Which is something you may want to reflect in a service registry.

So, to summarize, *annotations* are used to store more information about that resource and therefore is the closest concept to metadata, while *labels* are used to identify resources.

You can quickly annotate a resource, i.e. a service, like this:

```bash
kubectl annotate service service-name image-name=repo/name:tag
```

Similarly, remove an annotation as:

```bash
kubectl annotate service service-name image-name-
```

## Ownership

Whenever the CN-WAN Operator **creates** a resource - *any resource*, including namespaces, services and endpoints, on the service registry, it automatically inserts the reserved metadata `owner: cnwan-operator`. This will make the operator skip all those resources that have been created by someone else, i.e. manually by you or a program created by another entity: this will prevent us from messing up pre-existing configuration.

That being said, the operator will still insert child resources even if the parent resource is not owned by the operator. For example: if your service registry contains a service called `my-service` that does **not** have the `owner: cnwan-operator` metadata or that has something else entirely - i.e. `owner: someone-else`, then the operator will never update or delete its metadata, but will still add endpoints under it, as long as they, again, do not already exist and are owned by someone else.

Finally, if you wish the operator to manage your pre-existing resources on your service registry, please update all the necessary resources by inserting `owner: cnwan-operator` among their metadata.

## Enable namespaces

The CN-WAN Operator watches service updates only on *enabled* namespaces. To do so, you need to label a namespace with our reserved label key `operator.cnwan.io/enabled`.

If a namespace is labeled as `operator.cnwan.io/enabled=yes` then the operator will watch service updates happening on that namespace. On the contrary, `operator.cnwan.io/enabled=no` will instruct the operator to stay away from that namespace.

This being said, you don't need to rush labelling all namespaces as `operator.cnwan.io/enabled=no` in fear of potentially exposing sensitive data on the service registry: namespaces that do not have such label will be ignored by default, as the operator will pretend it is seeing `operator.cnwan.io/enabled=no`. This is useful in case you think you have few namespaces you want to enable or if you prefer to retain control, even have lots of namespaces to enable.

Instead, if you have many namespaces to enable and/or find it tedious to manually do so for every single one of them, you can override this behavior via `enableNamespaceByDefault: true` on the [operator settings](./configuration.md#enable-namespace-by-default): this means that the operator will pretend it is seeing `operator.cnwan.io/enabled=yes` and thus watch events in the namespace by default, unless instructed otherwise. This is the opposite scenario from above: now you will need to manually *disable* them.

Let's see some examples.

To _enable_ monitoring on namespace `hr`, do the following:

```bash
kubectl label ns hr operator.cnwan.io/enabled=yes
```

To _disable_ monitoring on namespace `hr`:

```bash
kubectl label ns hr operator.cnwan.io/enabled=no
```

Note: append `--overwrite` in case the label already exists.

## Allowed Annotations

As we said in [Metadata](#metadata), *annotations* are treated as metadata. To avoid publishing potentially sensitive data to the service registry, you can fine tune which annotations will be allowed and which will have to be ignored.

If a service does not have **at least** one of the allowed annotations, then it will be ignored by the operator or be removed from the service registry, if present.

You can define which annotations are allowed by setting up [configurations](./configuration.md#allow-annotations).

## Cloud Metadata

As the name suggests, *Cloud Metadata* are data that contain information about the Kubernetes cluster that is hosting the operator and the services that are going to be registered.
Such data can be the *Network*, *Subnetwork*, etc. The operator is able to retrieve some values automatically, depending on the Kubernetes platform, e.g. *GKE* or *EKS* but you can also provide some values manually through configuration.
These values will be stored in all registered services to be consumed by anyone interested in them, e.g. the CN-WAN Reader and the CN-WAN Adaptor.

To learn how to define them look at this [section](./configuration.md#cloud-metadata).

## Deploy

Please read our [installation guide](install.md).
