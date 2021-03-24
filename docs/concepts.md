# Concepts

## Table of Contents

* [How it Works](#how-it-works)
* [Supported Service Types](#supported-service-types)
* [Metadata](#metadata)
* [Annotations vs Labels](#annotations-vs-labels)
* [Ownership](#ownership)
* [Namespace Lists](#namespace-lists)
* [Namespace List Policy](#namespace-list-policy)
* [Allowed Annotations](#allowed-annotations)
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

## Namespace Lists

For the CN-WAN Operator, a namespace can belong to two lists: *allowlist* or *blocklist*. CN-WAN Operator only processes services that belong to namespaces it is allowed to work on: those that are inside the *allowlist*.

To insert a namespace in a list, you have to label it like this:

```bash
# Insert namespace in the allowlist
kubectl label ns namespace-name operator.cnwan.io/allowed=yes

# Insert namespace in the blocklist
kubectl label ns namespace-name operator.cnwan.io/blocked=yes
```

It doesn't really matter what you put as value (in this case `yes` has been inserted), just as long as the key `operator.cnwan.io/<key>` is as specified above.

Similarly, to remove a namespace from the list:

```bash
# Remove namespace from the allowlist
kubectl label ns namespace-name operator.cnwan.io/allowed-

# Remove namespace from the blocklist
kubectl label ns namespace-name operator.cnwan.io/blocked-
```

To prevent you from manually inserting a namespace in a list each time, you can define the [Default Namespace List Policy](#namespace-list-policy).

## Namespace List Policy

As we said, the operator watches for changes in Kubernetes services. While it does watch all services, it does **not** process services that belong to namespaces that the operator is not allowed to work on.

To prevent you from manually allowing/blocking namespaces each time, the operator defines a *default namespace list policy*.

Setting this default policy as `allowlist` means that, by default, all namespaces are **blocked** and the operator will work only on the ones you have specifically allowed. This is useful when you want few namespaces to be allowed or when you want to retain full control over which ones are allowed.

In contrast, if you have a lot of namespaces that you want to enable, or you want the operator to work more "automatically", or you virtually want to enable all namespaces, than you can set the default policy as `blocklist` which means that, by default, all namespaces are **allowed** and you will have to specify only the ones that must be blocked.

Refer to the previous section to know how to insert a namespace into a certain list.

Please follow [this guide](./configuration.md#set-the-namespace-list-policy) to learn how to set up the default namespace list policy.

## Allowed Annotations

As we said in [Metadata](#metadata), *annotations* are treated as metadata. To avoid publishing potentially sensitive data to the service registry, you can fine tune which annotations will be allowed and which will have to be ignored.

If a service does not have **at least** one of the allowed annotations, then it will be ignored by the operator or be removed from the service registry, if present.

You can define which annotations are allowed by setting up [configurations](./configuration.md#allow-annotations).

## Deploy

There are two ways to deploy the operator, according to your use case and knowledge of Kubernetes:

* If you want to use the operator "as-is", i.e. you don't want to change its code and/or resources, you can follow [Basic Installation](./basic_installation.md).
* If you want to modify resources, i.e. add new ones or update existing ones, you can follow [Advanced Installation](./advanced_installation.md).
